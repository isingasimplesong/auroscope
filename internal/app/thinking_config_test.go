package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThinkingConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, text, want string
		invalid          bool
	}{
		{name: "native default", text: `{}`},
		{name: "explicit", text: `{"thinking":"medium"}`, want: "medium"},
		{name: "no local enumeration", text: `{"thinking":"future-level"}`, want: "future-level"},
		{name: "empty", text: `{"thinking":""}`, invalid: true},
		{name: "blank", text: `{"thinking":"  "}`, invalid: true},
		{name: "null", text: `{"thinking":null}`, invalid: true},
		{name: "wrong type", text: `{"thinking":42}`, invalid: true},
		{name: "newline", text: `{"thinking":"high\nlow"}`, invalid: true},
		{name: "nul", text: `{"thinking":"high\u0000"}`, invalid: true},
		{name: "too long", text: `{"thinking":"` + strings.Repeat("a", 201) + `"}`, invalid: true},
		{name: "claude", text: `{"provider":"claude-code","thinking":"medium"}`, want: "medium"},
		{name: "api", text: `{"provider":"openai","model":"test","thinking":"medium"}`, want: "medium"},
		{name: "anthropic", text: `{"provider":"anthropic","model":"test","thinking":"medium"}`, want: "medium"},
		{name: "compatible", text: `{"provider":"openai-compatible","model":"test","thinking":"medium","base_url":"https://example.invalid/v1","api_key_env":"TEST_KEY"}`, want: "medium"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			writeModelConfig(t, dir, tc.text)
			c, err := loadAuditConfig()
			if (err != nil) != tc.invalid || !tc.invalid && c.Thinking != tc.want {
				t.Fatalf("thinking = %q, error = %v", c.Thinking, err)
			}
			if got := readString(t, filepath.Join(dir, "auroscope", "config.json")); got != tc.text {
				t.Fatal("existing configuration was rewritten")
			}
		})
	}
}

func TestCodexThinkingLiteralAndNativeDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	captured := filepath.Join(dir, "override")
	t.Setenv("THINKING_CAPTURE", captured)
	codex := writeExecutable(t, dir, "codex", `#!/bin/sh
set -eu
if [ "$1" = --version ]; then printf 'codex-cli 0.153.4\n'; exit 0; fi
out=''
override=''
count=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --config) shift; override=$1; count=$((count + 1)) ;;
    --output-last-message) shift; out=$1 ;;
  esac
  shift
done
[ "$count" -le 1 ]
printf '%s' "$override" > "$THINKING_CAPTURE"
printf '%s' '{"summary":"thinking received","risk":"low","findings":[],"uncertainty":"","inspect":[]}' > "$out"
`)
	for _, value := range []string{"", "high", `custom\level"; sandbox_mode="danger-full-access`, "future-level"} {
		fields := map[string]string{}
		want := ""
		if value != "" {
			fields["thinking"] = value
			quoted, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			want = "model_reasoning_effort=" + string(quoted)
		}
		data, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		writeModelConfig(t, dir, string(data))
		for _, direct := range []bool{false, true} {
			config := (runConfig{userConfig: true, codexPath: codex}).withDefaults()
			if direct {
				_, err = (codexClient{path: codex}).audit(auditBundle{}, config)
			} else {
				_, err = auditWithProvider(auditBundle{}, config)
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := readString(t, captured); got != want {
				t.Fatalf("override = %q, want %q", got, want)
			}
		}
	}
	writeModelConfig(t, dir, `{"thinking":null}`)
	if err := os.Remove(captured); err != nil {
		t.Fatal(err)
	}
	if _, err := auditWithProvider(auditBundle{}, (runConfig{codexPath: codex}).withDefaults()); err == nil {
		t.Fatal("invalid thinking accepted")
	}
	if _, err := os.Stat(captured); !os.IsNotExist(err) {
		t.Fatal("Codex executed with invalid thinking")
	}
}
