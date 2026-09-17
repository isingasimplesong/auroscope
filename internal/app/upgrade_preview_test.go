package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Also exercised against /usr/bin/auroscope by the installed-artifact gate.
func TestUpgradePreviewPassesThrough(t *testing.T) {
	for _, args := range [][]string{
		{"-Syu", "--print"},
		{"-Sup"},
		{"--sync", "--sysupgrade", "--print"},
		{"-Syu", "--print-format", "%n %v"},
		{"-Syu", "--print-format=%n %v"},
		{"-Syu", "--info"},
		{"-Syu", "--search", "hello"},
		{"-Syu", "--repo", "--print"},
	} {
		for _, wantStatus := range []int{0, 17} {
			t.Run(fmt.Sprintf("%s/status%d", strings.Join(args, "_"), wantStatus), func(t *testing.T) {
				dir := t.TempDir()
				calls := filepath.Join(dir, "calls")
				t.Setenv("CALLS", calls)
				t.Setenv("NATIVE_STATUS", fmt.Sprint(wantStatus))
				paru := writeExecutable(t, dir, "paru", `#!/bin/sh
printf 'CALL\n' >> "$CALLS"
printf '%s\n' "$@" >> "$CALLS"
printf 'native stdout\n'
printf 'native stderr\n' >&2
read -r answer
printf 'stdin=%s\n' "$answer"
exit "$NATIVE_STATUS"
`)
				marker := filepath.Join(dir, "codex-called")
				t.Setenv("CODEX_MARKER", marker)
				codex := writeExecutable(t, dir, "codex", "#!/bin/sh\n: > \"$CODEX_MARKER\"\nexit 99\n")
				t.Setenv("AUROSCOPE_PARU", paru)
				t.Setenv("AUROSCOPE_CODEX", codex)
				t.Setenv("AUROSCOPE_CLONE_DIR", filepath.Join(dir, "clones"))
				t.Setenv("AUROSCOPE_STATE", filepath.Join(dir, "state.sqlite3"))
				t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
				var stdout, stderr bytes.Buffer
				var status int
				if binary := os.Getenv("AUROSCOPE_TEST_BINARY"); binary != "" {
					cmd := exec.Command(binary, args...)
					cmd.Stdin, cmd.Stdout, cmd.Stderr = strings.NewReader("answer\n"), &stdout, &stderr
					status = commandStatus(cmd.Run())
				} else {
					status = run(args, runConfig{userConfig: true, stdin: strings.NewReader("answer\n"), stdout: &stdout, stderr: &stderr})
				}
				if status != wantStatus {
					t.Errorf("status = %d, want %d; stderr=%s", status, wantStatus, &stderr)
				}
				if got, want := readString(t, calls), "CALL\n"+strings.Join(args, "\n")+"\n"; got != want {
					t.Errorf("Paru invocations = %q, want exactly %q", got, want)
				}
				if stdout.String() != "native stdout\nstdin=answer\n" || stderr.String() != "native stderr\n" {
					t.Errorf("changed native streams: stdout=%q stderr=%q", &stdout, &stderr)
				}
				for _, path := range []string{marker, filepath.Join(dir, "clones"), filepath.Join(dir, "state.sqlite3"), filepath.Join(dir, "config", "auroscope")} {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Errorf("unexpected audit/preparation side effect at %s: %v", path, err)
					}
				}
			})
		}
	}
}
