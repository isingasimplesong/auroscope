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

// This matrix also runs against the installed package in the Arch gate.
func TestAuditStateBoundary(t *testing.T) {
	for _, mode := range []string{"explicit", "selection", "empty", "aur"} {
		for _, resource := range []string{"missing", "corrupt", "state-parent-file", "clone-file"} {
			for _, nativeStatus := range []int{0, 17, 130} {
				t.Run(fmt.Sprintf("%s/%s/status%d", mode, resource, nativeStatus), func(t *testing.T) {
					dir := t.TempDir()
					state, clones := filepath.Join(dir, "state.sqlite3"), filepath.Join(dir, "clones")
					sentinel := "not a database\n"
					preserved := ""
					switch resource {
					case "corrupt":
						preserved = state
					case "state-parent-file":
						preserved = filepath.Join(dir, "parent")
						state = filepath.Join(preserved, "state.sqlite3")
					case "clone-file":
						preserved = clones
					}
					if preserved != "" {
						if err := os.WriteFile(preserved, []byte(sentinel), 0600); err != nil {
							t.Fatal(err)
						}
					}
					calls := filepath.Join(dir, "calls")
					t.Setenv("CALLS", calls)
					t.Setenv("BOUNDARY_MODE", mode)
					t.Setenv("NATIVE_STATUS", fmt.Sprint(nativeStatus))
					paru := writeExecutable(t, dir, "paru", `#!/bin/sh
printf '%s\n' "$*" >> "$CALLS"
case "$1" in
 -Ssq) if [ "$BOUNDARY_MODE" != empty ]; then printf 'tree\n'; fi; exit 1 ;;
 -P) if [ "$BOUNDARY_MODE" = aur ]; then printf 'AUR TARGET tree tree\n'; else printf 'REPO TARGET extra tree\n'; fi; exit 0 ;;
esac
printf 'native stdout\n'
printf 'native stderr\n' >&2
read -r answer
printf 'answer=%s\n' "$answer"
exit "$NATIVE_STATUS"
`)
					marker := filepath.Join(dir, "provider-called")
					t.Setenv("PROVIDER_MARKER", marker)
					codex := writeExecutable(t, dir, "codex", "#!/bin/sh\n: > \"$PROVIDER_MARKER\"\nexit 99\n")
					t.Setenv("AUROSCOPE_PARU", paru)
					t.Setenv("AUROSCOPE_CODEX", codex)
					t.Setenv("AUROSCOPE_STATE", state)
					t.Setenv("AUROSCOPE_CLONE_DIR", clones)
					t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
					args := []string{"-S", "--needed", "tree"}
					expectedCalls := "-P --order tree\n-S --needed tree\n"
					if mode == "selection" || mode == "empty" {
						args = []string{"directory", "listing"}
						expectedCalls = "-Ssq --interactive directory listing\n-P --order tree\n-S -- tree\n"
					}
					if mode == "empty" {
						expectedCalls = "-Ssq --interactive directory listing\n"
					}
					// Valid AUR state proceeds into recipe acquisition, covered by existing
					// integration tests; this matrix checks only unusable AUR resources.
					if mode == "aur" && resource == "missing" {
						t.Skip("valid AUR path covered by integration tests")
					}
					input, err := os.CreateTemp(dir, "input")
					if err != nil {
						t.Fatal(err)
					}
					defer input.Close()
					if _, err := input.WriteString("yes\n"); err != nil {
						t.Fatal(err)
					}
					if _, err := input.Seek(0, 0); err != nil {
						t.Fatal(err)
					}
					var stdout, stderr bytes.Buffer
					var status int
					if binary := os.Getenv("AUROSCOPE_TEST_BINARY"); binary != "" {
						cmd := exec.Command(binary, args...)
						cmd.Stdin, cmd.Stdout, cmd.Stderr = input, &stdout, &stderr
						status = commandStatus(cmd.Run())
					} else {
						status = run(args, runConfig{userConfig: true, stdin: input, stdout: &stdout, stderr: &stderr})
					}
					switch mode {
					case "aur":
						expectedCalls = "-P --order tree\n"
						if status != 1 || !(strings.Contains(stderr.String(), "open state:") || strings.Contains(stderr.String(), "prepare clone directory:")) {
							t.Errorf("AUR did not fail closed: %d %q", status, &stderr)
						}
					case "empty":
						if status != 0 || stdout.Len() != 0 || stderr.String() != "auroscope: no AUR targets selected\n" {
							t.Errorf("empty selection: %d %q %q", status, &stdout, &stderr)
						}
					default:
						if status != nativeStatus || stdout.String() != "native stdout\nanswer=yes\n" || stderr.String() != "native stderr\n" {
							t.Errorf("native behavior changed: %d %q %q", status, &stdout, &stderr)
						}
					}
					if got := readString(t, calls); got != expectedCalls {
						t.Errorf("calls=%q, want %q", got, expectedCalls)
					}
					if preserved != "" && readString(t, preserved) != sentinel {
						t.Error("existing audit resource modified")
					}
					absent := []string{marker, filepath.Join(dir, "config", "auroscope")}
					if mode != "aur" {
						if resource != "clone-file" {
							absent = append(absent, clones)
						}
						if resource == "missing" || resource == "clone-file" {
							absent = append(absent, state)
						}
						absent = append(absent, state+"-wal", state+"-shm")
					}
					for _, path := range absent {
						if _, err := os.Stat(path); err == nil {
							t.Errorf("unexpected audit side effect: %s", path)
						}
					}
				})
			}
		}
	}
}
