package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfiguredPromptReachesCodex(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "auroscope", "config.json")
	if _, err := loadAuditPrompt(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"prompt":"Custom packaging audit"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	codex := writeExecutable(t, dir, "codex", `#!/bin/sh
set -eu
if [ "$1" = --version ]; then
  printf 'codex-cli 0.150.1\n'
  exit 0
fi
out=''
while [ "$#" -gt 0 ]; do
  if [ "$1" = --output-last-message ]; then shift; out=$1; fi
  last=$1
  shift
done
[ "$last" = 'Custom packaging audit' ]
printf '%s' '{"summary":"custom prompt received","risk":"low","findings":[],"uncertainty":"","inspect":[]}' > "$out"
`)
	if _, err := (codexClient{path: codex}).audit(auditBundle{}, runConfig{userConfig: true}); err != nil {
		t.Fatal(err)
	}
}

func TestAuditConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auroscope", "config.json")
	prompt, err := loadAuditPrompt(path)
	if err != nil || prompt != auditPrompt() {
		t.Fatalf("initial prompt = %q, %v", prompt, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("configuration permissions: %v, %v", info, err)
	}
	for i, text := range []string{
		`{"prompt":"Custom packaging audit.\nReturn the supplied JSON schema."}`,
		`{"prompt":""}`,
		`{"prompt":" \n	"}`,
		`{}`,
		`{"prompt":null}`,
		`{broken`,
	} {
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		prompt, err := loadAuditPrompt(path)
		if i == 0 {
			if err != nil || prompt != "Custom packaging audit.\nReturn the supplied JSON schema." {
				t.Fatalf("custom prompt = %q, %v", prompt, err)
			}
		} else if err == nil {
			t.Fatal("invalid configuration accepted")
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != text {
			t.Fatalf("existing configuration overwritten: %q, %v", got, err)
		}
	}
}
