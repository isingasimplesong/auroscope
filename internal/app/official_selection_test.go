package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOfficialSelectionFinalInstall(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{"search", []string{"tree"}, []string{"-Ssq --interactive tree", "-P --order tree", "-S -- tree"}},
		{"search terms differ from target", []string{"directory", "listing"}, []string{"-Ssq --interactive directory listing", "-P --order tree", "-S -- tree"}},
		{"explicit options", []string{"-S", "--needed", "--noconfirm", "extra/tree"}, []string{"-P --order extra/tree", "-S --needed --noconfirm extra/tree"}},
		{"explicit delimiter", []string{"--sync", "--needed", "--", "tree"}, []string{"-P --order tree", "--sync --needed -- tree"}},
	} {
		for _, nativeStatus := range []int{0, 17, 130} {
			t.Run(fmt.Sprintf("%s/status%d", tc.name, nativeStatus), func(t *testing.T) {
				dir := t.TempDir()
				calls := filepath.Join(dir, "calls")
				marker := filepath.Join(dir, "codex-called")
				t.Setenv("CALLS", calls)
				t.Setenv("CODEX_MARKER", marker)
				t.Setenv("NATIVE_STATUS", fmt.Sprint(nativeStatus))
				paru := writeExecutable(t, dir, "paru", `#!/bin/sh
printf '%s\n' "$*" >> "$CALLS"
case "$1" in
  -Ssq) printf 'tree\n'; exit 1 ;;
  -P) printf 'REPO TARGET extra tree\n'; exit 0 ;;
esac
printf 'native stdout\n'
printf 'native stderr\n' >&2
IFS= read -r answer
printf 'answer=%s\n' "$answer"
exit "$NATIVE_STATUS"
`)
				codex := writeExecutable(t, dir, "codex", `#!/bin/sh
: > "$CODEX_MARKER"
exit 99
`)
				var stdout, stderr bytes.Buffer
				// A file preserves unread native input across the selection child;
				// os/exec would eagerly drain a strings.Reader into its pipe.
				inputPath := filepath.Join(dir, "input")
				if err := os.WriteFile(inputPath, []byte("yes\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				input, err := os.Open(inputPath)
				if err != nil {
					t.Fatal(err)
				}
				defer input.Close()
				status := run(tc.args, runConfig{
					paruPath: paru, codexPath: codex,
					statePath: filepath.Join(dir, "state.sqlite3"), cloneDir: filepath.Join(dir, "clones"),
					stdin: input, stdout: &stdout, stderr: &stderr,
				})
				if status != nativeStatus {
					t.Errorf("status = %d, want %d; stderr = %q", status, nativeStatus, stderr.String())
				}
				if got := strings.Split(strings.TrimSpace(readString(t, calls)), "\n"); !reflect.DeepEqual(got, tc.want) {
					t.Errorf("calls = %#v, want %#v", got, tc.want)
				}
				if stdout.String() != "native stdout\nanswer=yes\n" || stderr.String() != "native stderr\n" {
					t.Errorf("native streams changed: stdout=%q stderr=%q", stdout.String(), stderr.String())
				}
				if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("Codex was invoked or marker stat failed: %v", err)
				}
			})
		}
	}
}
