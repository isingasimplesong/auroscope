package app

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/creack/pty"
)

// Run only inside the pinned disposable-Arch gate; never access host Pacman.
func TestSelectionRealParuColors(t *testing.T) {
	if os.Getenv("AUROSCOPE_E2E_INSIDE") != "1" || os.Getenv("AUROSCOPE_TEST_REAL_PARU") == "" {
		t.Skip("requires disposable Arch and pinned Paru")
	}
	original, err := os.ReadFile("/etc/pacman.conf")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name            string
		color, terminal bool
	}{
		{"color terminal", true, true}, {"no color terminal", false, true}, {"color redirected", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			var lines []string
			for _, line := range strings.Split(string(original), "\n") {
				if strings.TrimSpace(line) == "Color" {
					continue
				}
				lines = append(lines, line)
				if strings.TrimSpace(line) == "[options]" && tc.color {
					lines = append(lines, "Color")
				}
			}
			configPath := filepath.Join(dir, "pacman.conf")
			if err := os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AUROSCOPE_TEST_PACMAN_CONF", configPath)
			wrapper := writeExecutable(t, dir, "paru", `#!/bin/sh
# For the installed-artifact probe, stop after selection with a distinctive
# status. Verify the exact machine target reached planning, without a recipe.
if [ "${AUROSCOPE_TEST_BINARY:-}" != "" ] && [ "${1:-}" = '-P' ]; then
  [ "$#" = 3 ] && [ "$2" = '--order' ] && [ "$3" = 'hello' ] || exit 74
  exit 73
fi
exec "$AUROSCOPE_TEST_REAL_PARU" --config "$AUROSCOPE_TEST_PACMAN_CONF" "$@"
`)
			var output bytes.Buffer
			var stdout, stderr io.Writer = io.Discard, &output
			master, slave, err := pty.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer slave.Close()
			terminalOutput := make(chan []byte, 1)
			go func() { data, _ := io.ReadAll(master); terminalOutput <- data }()
			if tc.terminal {
				stdout, stderr = slave, slave
			}
			client := paruClient{config: runConfig{paruPath: wrapper, stdin: strings.NewReader("1\n"), stdout: stdout, stderr: stderr}}
			var targets []string
			if binary := os.Getenv("AUROSCOPE_TEST_BINARY"); binary != "" {
				cmd := exec.Command(binary, "hello")
				cmd.Env = append(os.Environ(), "AUROSCOPE_PARU="+wrapper)
				cmd.Stdin, cmd.Stdout, cmd.Stderr = strings.NewReader("1\n"), stdout, stderr
				if exit, ok := cmd.Run().(*exec.ExitError); !ok || exit.ExitCode() != 73 {
					t.Errorf("installed selection did not reach exact-target planning: %v", exit)
				}
			} else {
				targets, err = client.selectPackages([]string{"hello"})
			}
			slave.Close()
			output.Write(<-terminalOutput)
			if err != nil {
				t.Fatalf("selection: %v; menu %q", err, output.String())
			}
			if os.Getenv("AUROSCOPE_TEST_BINARY") == "" && (len(targets) != 1 || unqualifiedTarget(targets[0]) != "hello") {
				t.Fatalf("targets = %q", targets)
			}
			if got := bytes.Contains(output.Bytes(), []byte("\x1b[")); got != (tc.color && tc.terminal) {
				t.Fatalf("color=%v, menu=%q", got, output.String())
			}
			if !strings.Contains(output.String(), "Select packages") {
				t.Fatalf("missing native prompt: %q", output.String())
			}
		})
	}
}
