package app

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditThinkingConfiguration(t *testing.T) {
	for _, tc := range []struct {
		content string
		invalid bool
	}{
		{`{}`, false},
		{`{"thinking":"medium"}`, false},
		{`{"thinking":"future-level"}`, false},
		{`{"thinking":""}`, false},
		{`{"thinking":"  medium  "}`, false},
		{`{"thinking":null}`, true},
		{`{"thinking":42}`, true},
		{`{"thinking":true}`, true},
		{`{"thinking":[]}`, true},
		{`{"provider":"claude-code","thinking":"medium"}`, false},
		{`{"provider":"openai","model":"m","thinking":"medium"}`, false},
		{`{"provider":"claude-code","thinking":""}`, true},
		{`{"provider":"openai","model":"m","thinking":"high\nlow"}`, true},
	} {
		t.Run(tc.content, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			writeModelConfig(t, dir, tc.content)
			got, err := loadAuditConfig()
			if (err != nil) != tc.invalid {
				t.Fatalf("config error = %v", err)
			}
			if !tc.invalid {
				var want auditConfig
				if err := json.Unmarshal([]byte(tc.content), &want); err != nil {
					t.Fatal(err)
				}
				if (got.Thinking == nil) != (want.Thinking == nil) || want.Thinking != nil && *got.Thinking != *want.Thinking {
					t.Fatal("thinking string was changed")
				}
			}
			if readString(t, filepath.Join(dir, "auroscope", "config.json")) != tc.content {
				t.Fatal("configuration was rewritten")
			}
		})
	}
}

func TestCodexReceivesConfiguredThinking(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	codex := writeExecutable(t, dir, "codex", `#!/bin/sh
set -eu
if [ "$1" = --version ]; then
  printf 'codex-cli 0.153.4\n'
  exit 0
fi
value=''
count=0
out=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    --config) shift; value=$1; count=$((count + 1)) ;;
    --output-last-message) shift; out=$1 ;;
  esac
  shift
done
[ "$value" = "$EXPECTED_THINKING" ]
[ "$count" = "$EXPECTED_COUNT" ]
printf '%s' '{"summary":"thinking verified","risk":"low","findings":[],"uncertainty":"","inspect":[]}' >"$out"
`)
	for _, direct := range []bool{false, true} {
		for _, value := range []string{"medium", "future-level", "", "  medium  ", "x\"\nmodel=\"other", "true", "\x00", "omitted"} {
			data := `{}`
			want, count := "", "0"
			if value != "omitted" {
				encoded, _ := json.Marshal(value)
				data = `{"thinking":` + string(encoded) + `}`
				want, count = "model_reasoning_effort="+string(encoded), "1"
			}
			writeModelConfig(t, dir, data)
			t.Setenv("EXPECTED_THINKING", want)
			t.Setenv("EXPECTED_COUNT", count)
			config := (runConfig{userConfig: true, codexPath: codex, stdout: io.Discard}).withDefaults()
			var err error
			if direct {
				_, err = (codexClient{path: codex}).audit(auditBundle{}, config)
			} else {
				_, err = auditWithProvider(auditBundle{}, config)
			}
			if err != nil {
				t.Fatalf("direct=%v value=%q: %v", direct, value, err)
			}
		}
	}
	writeModelConfig(t, dir, `{"thinking":42}`)
	paru := writeExecutable(t, dir, "paru", "#!/bin/sh\nexit 37\n")
	if got := run([]string{"-S", "--repo", "tree"}, runConfig{
		userConfig: true, paruPath: paru, stdin: strings.NewReader(""), stdout: io.Discard, stderr: io.Discard,
	}); got != 37 {
		t.Fatalf("official exit status = %d", got)
	}
}
