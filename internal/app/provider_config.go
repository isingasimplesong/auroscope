package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The shared user file is read only at the audit boundary, never on official
// operations. Model/prompt follow-up work must extend this file, not add another.
type auditConfig struct {
	Provider  string  `json:"provider"`
	Model     string  `json:"model,omitempty"`
	Thinking  string  `json:"thinking,omitempty"`
	Prompt    *string `json:"prompt,omitempty"`
	BaseURL   string  `json:"base_url,omitempty"`
	APIKeyEnv string  `json:"api_key_env,omitempty"`
}

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func loadAuditConfig() (auditConfig, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return auditConfig{}, errors.New("cannot locate AURoscope configuration directory")
	}
	path := filepath.Join(root, "auroscope", "config.json")
	return loadAuditConfigPath(path)
}

func loadAuditConfigPath(path string) (auditConfig, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := createAuditConfig(path); err != nil {
			return auditConfig{}, err
		}
		f, err = os.Open(path)
	}
	if err != nil {
		return auditConfig{}, errors.New("cannot read auroscope/config.json")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 64*1024+1))
	if err != nil || len(data) > 64*1024 {
		return auditConfig{}, errors.New("AURoscope configuration is unreadable or exceeds 64 KiB")
	}
	var c auditConfig
	if validateJSONDocument(data) != nil || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return c, errors.New("invalid AURoscope configuration JSON")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return c, errors.New("AURoscope configuration must be a JSON object")
	}
	for _, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return c, errors.New("audit configuration fields cannot be null")
		}
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	// Do not echo JSON parser errors: the user may have pasted a secret value.
	if err := d.Decode(&c); err != nil {
		return c, errors.New("invalid audit configuration: expected provider, model, thinking, prompt, base_url and api_key_env only")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return c, errors.New("invalid AURoscope configuration: trailing data")
	}
	if _, present := fields["model"]; present && strings.TrimSpace(c.Model) == "" {
		return c, errors.New("invalid AURoscope configuration: model must be non-empty")
	}
	if _, present := fields["thinking"]; present && strings.TrimSpace(c.Thinking) == "" {
		return c, errors.New("invalid AURoscope configuration: thinking must be non-empty")
	}
	return c.normalized()
}

func (c auditConfig) normalized() (auditConfig, error) {
	if c.Prompt != nil && strings.TrimSpace(*c.Prompt) == "" {
		return c, errors.New("audit configuration requires a nonempty prompt")
	}
	if c.Provider == "" {
		c.Provider = "codex"
	}
	if c.Provider == "codex" && c.Model == "" {
		c.Model = "gpt-5.6-luna"
	}
	if len(c.Model) > 200 || strings.ContainsAny(c.Model, "\r\n\x00") {
		return c, errors.New("invalid audit model")
	}
	if len(c.Thinking) > 200 || strings.ContainsAny(c.Thinking, "\r\n\x00") {
		return c, errors.New("invalid audit thinking value")
	}

	switch c.Provider {
	case "codex", "claude-code":
		if c.BaseURL != "" || c.APIKeyEnv != "" {
			return c, errors.New("CLI providers use native authentication, not base_url or api_key_env")
		}
	case "anthropic", "openai", "openai-compatible":
		if strings.TrimSpace(c.Model) == "" {
			return c, errors.New("the selected API provider requires an explicit model")
		}
		if c.Provider != "openai-compatible" && c.BaseURL != "" {
			return c, errors.New("custom base_url requires the openai-compatible provider")
		}
		if c.Provider == "anthropic" {
			c.BaseURL = "https://api.anthropic.com/v1"
		}
		if c.Provider == "openai" {
			c.BaseURL = "https://api.openai.com/v1"
		}
		if c.APIKeyEnv == "" {
			switch c.Provider {
			case "anthropic":
				c.APIKeyEnv = "ANTHROPIC_API_KEY"
			case "openai":
				c.APIKeyEnv = "OPENAI_API_KEY"
			default:
				return c, errors.New("openai-compatible requires api_key_env")
			}
		}
		if !environmentName.MatchString(c.APIKeyEnv) {
			return c, errors.New("api_key_env must name an environment variable, not contain a key")
		}
		u, err := url.Parse(c.BaseURL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
			return c, errors.New("base_url must be an HTTPS API root without credentials, query or fragment")
		}
		c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	default:
		return c, errors.New("unknown audit provider; choose codex, claude-code, anthropic, openai or openai-compatible")
	}
	return c, nil
}

func (c auditConfig) label() string {
	switch c.Provider {
	case "", "codex":
		return "Codex"
	case "claude-code":
		return "Claude Code"
	case "anthropic":
		return "Anthropic API"
	case "openai":
		return "OpenAI API"
	case "openai-compatible":
		return "OpenAI-compatible API"
	default:
		return "configured provider"
	}
}

func auditWithProvider(bundle auditBundle, config runConfig) (auditReport, error) {
	c, err := loadAuditConfig()
	if err != nil {
		return auditReport{}, err
	}
	fmt.Fprintf(config.stdout, "AURoscope: auditing %s with %s (timeout %s)...\n", escapeTerminal(bundle.Identity.Pkgbase), c.label(), config.codexTimeout)
	config.auditPrompt = c.prompt()
	switch c.Provider {
	case "codex":
		config.auditModel = c.Model
		config.auditThinking = c.Thinking
		return (codexClient{path: config.codexPath}).audit(bundle, config)
	case "claude-code":
		return auditClaude(bundle, c, config)
	default:
		return auditHTTP(bundle, c, config, nil)
	}
}
