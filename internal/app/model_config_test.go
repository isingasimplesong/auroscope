package app

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditModelConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, content, want string
		missing, invalid    bool
	}{
		{name: "absent", missing: true, want: "luna"},
		{name: "omitted", content: `{}`, want: "luna"},
		{name: "codex provider", content: `{"provider":"codex"}`, want: "luna"},
		{name: "shared provider and model", content: `{"provider":"codex","model":"chosen"}`, want: "chosen"},
		{name: "claude native default", content: `{"provider":"claude-code"}`, want: ""},
		{name: "api explicit model", content: `{"provider":"openai","model":"api-model"}`, want: "api-model"},
		{name: "api requires model", content: `{"provider":"openai"}`, invalid: true},
		{name: "custom", content: `{"model":"another-model"}`, want: "another-model"},
		{name: "shared prompt and model", content: `{"model":"chosen","prompt":"custom audit"}`, want: "chosen"},
		{name: "literal", content: `{"model":"custom model;not-a-shell"}`, want: "custom model;not-a-shell"},
		{name: "empty", content: `{"model":""}`, invalid: true},
		{name: "whitespace", content: `{"model":"  "}`, invalid: true},
		{name: "null model", content: `{"model":null}`, invalid: true},
		{name: "null object", content: `null`, invalid: true},
		{name: "wrong type", content: `{"model":42}`, invalid: true},
		{name: "unknown key", content: `{"modle":"other"}`, invalid: true},
		{name: "broken", content: `{`, invalid: true},
		{name: "trailing", content: `{} {}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			if !tc.missing {
				writeModelConfig(t, dir, tc.content)
			}
			got, err := loadAuditConfig()
			if (err != nil) != tc.invalid || !tc.invalid && got.Model != tc.want {
				t.Fatalf("model = %q, error = %v", got.Model, err)
			}
			if tc.missing && (got.Prompt == nil || *got.Prompt != auditPrompt()) {
				t.Fatal("missing config must create the full default prompt alongside the luna default")
			}
			if !tc.missing {
				data, readErr := os.ReadFile(filepath.Join(dir, "auroscope", "config.json"))
				if readErr != nil || string(data) != tc.content {
					t.Fatal("model selection must not rewrite existing configuration")
				}
			}
		})
	}
}

func writeModelConfig(t *testing.T, dir, content string) {
	t.Helper()
	dir = filepath.Join(dir, "auroscope")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAuditModelHomeFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeModelConfig(t, filepath.Join(home, ".config"), `{"model":"home-model"}`)
	if got, err := loadAuditConfig(); err != nil || got.Model != "home-model" {
		t.Fatalf("model = %q, error = %v", got.Model, err)
	}
}

func TestCodexReceivesConfiguredModel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	codex := writeExecutable(t, dir, "codex", `#!/bin/sh
set -eu
if [ "$1" = --version ]; then
  printf 'codex-cli 0.153.4\n'
  exit 0
fi
model=''
out=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    --model) shift; model=$1 ;;
    --output-last-message) shift; out=$1 ;;
  esac
  shift
done
[ "$model" = "$EXPECTED_MODEL" ]
printf '%s' '{"summary":"model verified","risk":"low","findings":[],"uncertainty":"","inspect":[]}' >"$out"
`)
	for _, model := range []string{"luna", "custom model;not-a-shell"} {
		t.Setenv("EXPECTED_MODEL", model)
		if model != "luna" {
			writeModelConfig(t, dir, `{"model":"custom model;not-a-shell"}`)
		}
		if _, err := auditWithProvider(auditBundle{}, (runConfig{codexPath: codex}).withDefaults()); err != nil {
			t.Fatal(err)
		}
	}
	writeModelConfig(t, dir, `{"model":""}`)
	if _, err := auditWithProvider(auditBundle{}, (runConfig{codexPath: "/does/not/exist"}).withDefaults()); err == nil || !strings.Contains(err.Error(), "invalid AURoscope configuration") {
		t.Fatalf("want configuration rejection before invoking Codex, got %v", err)
	}
	// A retry must reread a corrected configuration, not cache the failure.
	writeModelConfig(t, dir, `{"model":"retry-model"}`)
	t.Setenv("EXPECTED_MODEL", "retry-model")
	if _, err := auditWithProvider(auditBundle{}, (runConfig{codexPath: codex}).withDefaults()); err != nil {
		t.Fatal(err)
	}
}

func TestOfficialOperationIgnoresInvalidModelConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	writeModelConfig(t, dir, `{"model":""}`)
	paru := writeExecutable(t, dir, "paru", "#!/bin/sh\nexit 37\n")
	if got := run([]string{"-S", "--repo", "tree"}, runConfig{
		paruPath: paru, stdin: strings.NewReader(""), stdout: io.Discard, stderr: io.Discard,
	}); got != 37 {
		t.Fatalf("official exit status = %d, want 37", got)
	}
}
