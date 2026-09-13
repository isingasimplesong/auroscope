package app

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestProviderInstalledHTTP(t *testing.T) {
	binary := os.Getenv("AUROSCOPE_TEST_BINARY")
	if binary == "" {
		binary = filepath.Join(t.TempDir(), "auroscope")
		cmd := exec.Command("go", "build", "-o", binary, "../../cmd/auroscope")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build: %v %s", err, out)
		}
	}
	for _, success := range []bool{true, false} {
		t.Run(map[bool]string{true: "approve", false: "failed-skip"}[success], func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("TEST_API_KEY", "credential-fixture")
			var requests atomic.Int32
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				data, _ := io.ReadAll(r.Body)
				if strings.Contains(string(data), "credential-fixture") {
					t.Error("credential reached model input")
				}
				if r.Header.Get("Authorization") != "Bearer credential-fixture" {
					t.Error("missing auth")
				}
				var request map[string]any
				if json.Unmarshal(data, &request) != nil || request["reasoning_effort"] != "medium" {
					t.Error("configured thinking did not reach API")
				}
				if !success {
					w.WriteHeader(401)
					io.WriteString(w, "credential-fixture")
					return
				}
				io.WriteString(w, providerEnvelope("openai-compatible", providerTestReport))
			}))
			defer s.Close()
			caPath := filepath.Join(dir, "fixture-ca.pem")
			if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.Certificate().Raw}), 0o600); err != nil {
				t.Fatal(err)
			}
			thinking := "medium"
			data, _ := json.Marshal(auditConfig{Provider: "openai-compatible", Model: "fixture-model", Thinking: &thinking, BaseURL: s.URL + "/v1", APIKeyEnv: "TEST_API_KEY"})
			writeProviderConfig(t, string(data))
			repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\n")
			calls := filepath.Join(dir, "calls")
			paru := fakeParu(t, dir, calls, repo, "AUR TARGET hello hello\n")
			state := filepath.Join(dir, "state.sqlite3")
			cmd := exec.Command(binary, "-S", "hello")
			cmd.Env = append(os.Environ(), "SSL_CERT_FILE="+caPath,
				"AUROSCOPE_PARU="+paru, "AUROSCOPE_CODEX=/does/not/exist",
				"AUROSCOPE_STATE="+state, "AUROSCOPE_CLONE_DIR="+filepath.Join(dir, "clones"))
			input := "approve\n"
			if !success {
				input = "skip\n"
			}
			cmd.Stdin = strings.NewReader(input)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("binary: %v %s", err, out)
			}
			if requests.Load() != 1 || strings.Contains(string(out), "credential-fixture") {
				t.Fatalf("request count=%d or leaked key", requests.Load())
			}
			if strings.Contains(readString(t, calls), "--skipreview") != success {
				t.Fatal("incorrect final handoff")
			}
			if data, err := os.ReadFile(state); err != nil || bytes.Contains(data, []byte("credential-fixture")) {
				t.Fatal("credential in state or state unreadable")
			}
		})
	}
}

// A genuine live smoke is explicitly opt-in; absent credentials are not a pass.
// Send only inert metadata, never a real local recipe or host configuration.
func TestProviderLiveAudit(t *testing.T) {
	name := os.Getenv("AUROSCOPE_TEST_LIVE_PROVIDER")
	if name == "" {
		t.Skip("set AUROSCOPE_TEST_LIVE_PROVIDER plus model and native/API authentication to qualify one provider")
	}
	c, err := (auditConfig{
		Provider: name, Model: os.Getenv("AUROSCOPE_TEST_LIVE_MODEL"),
		BaseURL: os.Getenv("AUROSCOPE_TEST_LIVE_BASE_URL"), APIKeyEnv: os.Getenv("AUROSCOPE_TEST_LIVE_KEY_ENV"),
	}).normalized()
	if err != nil {
		t.Fatal(err)
	}
	bundle := auditBundle{Label: "UNTRUSTED inert packaging metadata", Identity: recipeIdentity{Pkgbase: "provider-contract"}, Mode: "full", Files: []recipeFile{{Path: "PKGBUILD", Text: "pkgname=provider-contract\npkgver=1\npkgrel=1\n"}}}
	config := runConfig{stdout: io.Discard, stderr: io.Discard}.withDefaults()
	switch c.Provider {
	case "claude-code":
		_, err = auditClaude(bundle, c, config)
	case "codex":
		config.auditModel = c.Model
		_, err = (codexClient{path: config.codexPath}).audit(bundle, config)
	default:
		_, err = auditHTTP(bundle, c, config, nil)
	}
	if err != nil {
		t.Fatal("live provider audit failed; no qualification claimed")
	}
	t.Log("actual production client returned a locally validated report; no package code executed")
}
