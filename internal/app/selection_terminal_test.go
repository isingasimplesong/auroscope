package app

import (
	"bytes"
	"io"

	"reflect"
	"strings"
	"testing"

	"github.com/creack/pty"
)

func TestSelectionPreservesNativeColorDetection(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		stdoutTTY, stderrTTY bool
	}{
		{"terminal", true, true}, {"redirected stdout", false, true}, {"redirected stderr", true, false}, {"redirected both", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			master, slave, err := pty.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer slave.Close()
			var redirected bytes.Buffer
			var stdout io.Writer = &redirected
			var stderr io.Writer = &redirected
			if tc.stdoutTTY {
				stdout = slave
			}
			if tc.stderrTTY {
				stderr = slave
			}
			paru := writeExecutable(t, t.TempDir(), "paru", `#!/bin/sh
if test -t 1 && test -t 2; then
  printf '\033[32mnative menu\033[0m\n' >&2
else
  printf 'native menu\n' >&2
fi
printf 'aur/hello\nworld-bin\n'
exit 1
`)
			client := paruClient{config: runConfig{paruPath: paru, stdin: strings.NewReader(""), stdout: stdout, stderr: stderr}}
			selected, err := client.selectPackages([]string{"hello"})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(selected, []string{"aur/hello", "world-bin"}) {
				t.Fatalf("targets = %q", selected)
			}
			slave.Close()
			terminalOutput, _ := io.ReadAll(master)
			output := string(terminalOutput) + redirected.String()
			if got := strings.Contains(output, "\x1b[32m"); got != (tc.stdoutTTY && tc.stderrTTY) {
				t.Fatalf("native color = %v; output = %q", got, output)
			}
			if strings.Contains(output, "aur/hello") {
				t.Fatalf("machine targets leaked: %q", output)
			}
		})
	}
}

func TestSelectionTerminalStartFailure(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	client := paruClient{config: runConfig{paruPath: "/nonexistent/paru", stdin: strings.NewReader(""), stdout: slave, stderr: slave}}
	if _, err := client.selectPackages([]string{"hello"}); err == nil {
		t.Fatal("missing executable accepted")
	}
}
