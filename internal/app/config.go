package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loadAuditPrompt creates user configuration only when an AUR audit needs it.
// Existing configuration is never rewritten, including during upgrades.
func loadAuditPrompt(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return "", fmt.Errorf("create audit configuration directory: %w", err)
		}
		defaults, marshalErr := json.MarshalIndent(struct {
			Prompt string `json:"prompt"`
		}{auditPrompt()}, "", "  ")
		if marshalErr != nil {
			return "", marshalErr
		}
		file, createErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if createErr == nil {
			_, writeErr := file.Write(append(defaults, '\n'))
			closeErr := file.Close()
			if writeErr != nil {
				return "", fmt.Errorf("write audit configuration: %w", writeErr)
			}
			if closeErr != nil {
				return "", closeErr
			}
		} else if !os.IsExist(createErr) {
			return "", fmt.Errorf("create audit configuration: %w", createErr)
		}
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("read audit configuration %s: %w", path, err)
	}
	var config struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("invalid audit configuration %s: %w", path, err)
	}
	if strings.TrimSpace(config.Prompt) == "" {
		return "", fmt.Errorf("audit configuration %s requires a nonempty prompt", path)
	}
	return config.Prompt, nil
}
