package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfficialOperationDoesNotTouchAuditConfiguration(t *testing.T) {
	for _, existing := range []bool{false, true} {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)
		path := filepath.Join(dir, "auroscope", "config.json")
		if existing {
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("invalid JSON"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		paru := writeExecutable(t, dir, "paru", "#!/bin/sh\nprintf 'native output\\n'\nexit 17\n")
		var output bytes.Buffer
		status := run([]string{"-S", "--repo", "tree"}, runConfig{
			userConfig: true, paruPath: paru, codexPath: "/does/not/exist",
			stdin: strings.NewReader(""), stdout: &output, stderr: io.Discard,
		})
		if status != 17 || output.String() != "native output\n" {
			t.Fatalf("native result changed: %d, %q", status, output.String())
		}
		data, err := os.ReadFile(path)
		if existing {
			if err != nil || string(data) != "invalid JSON" {
				t.Fatalf("existing config changed: %q, %v", data, err)
			}
		} else if !os.IsNotExist(err) {
			t.Fatalf("official operation created configuration: %v", err)
		}
	}
}

func TestInvalidPromptCanSkipWithoutAuditOrBuild(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "auroscope", "config.json")
	if _, err := loadAuditPrompt(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"prompt":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\n")
	calls := filepath.Join(dir, "calls")
	paru := fakeParu(t, dir, calls, repo, "AUR TARGET hello hello\n")
	marker := filepath.Join(dir, "codex-exec")
	t.Setenv("CODEX_EXEC_MARKER", marker)
	codex := writeExecutable(t, dir, "codex", `#!/bin/sh
if [ "$1" = --version ]; then
  printf 'codex-cli 0.150.1\n'
  exit 0
fi
: > "$CODEX_EXEC_MARKER"
exit 99
`)
	var output, diagnostics bytes.Buffer
	status := run([]string{"-S", "hello"}, runConfig{
		userConfig: true, paruPath: paru, codexPath: codex,
		statePath: filepath.Join(dir, "state.sqlite3"), cloneDir: filepath.Join(dir, "clones"),
		stdin: strings.NewReader(""), reviewInput: strings.NewReader("skip\n"),
		stdout: &output, stderr: &diagnostics,
	})
	if status != 0 || !strings.Contains(diagnostics.String(), "requires a nonempty prompt") {
		t.Fatalf("status=%d, stdout=%s, stderr=%s", status, &output, &diagnostics)
	}
	if !strings.Contains(output.String(), "[r]etry | [s]kip | [c]ancel") {
		t.Fatalf("missing recovery choices: %s", &output)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("Codex exec ran with an invalid prompt: %v", err)
	}
	if strings.Contains(readString(t, calls), "--skipreview") {
		t.Fatal("invalid prompt allowed a final build")
	}
}
