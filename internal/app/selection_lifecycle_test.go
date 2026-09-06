package app

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestSelectionWithControllingTerminal(t *testing.T) {
	paru := writeExecutable(t, t.TempDir(), "paru", `#!/bin/sh
test -t 0 && test -t 1 && test -t 2 || exit 91
printf '\033[32mChoose packages: \033[0m' >&2
IFS= read -r answer
test "$answer" = 1 || exit 92
printf 'aur/hello\n'
exit 1
`)
	cmd := exec.Command(os.Args[0], "-test.run=^TestSelectionPTYHelper$", "-test.timeout=10s")
	cmd.Env = append(os.Environ(), "AUROSCOPE_TEST_SELECTION_HELPER="+paru)
	master, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	if _, err := io.WriteString(master, "1\n"); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	output, _ := io.ReadAll(master)
	if err != nil {
		t.Fatalf("helper: %v; %q", err, output)
	}
	if !bytes.Contains(output, []byte("\x1b[32mChoose packages:")) {
		t.Fatalf("lost menu color: %q", output)
	}
	if bytes.Contains(output, []byte("aur/hello")) {
		t.Fatalf("leaked targets: %q", output)
	}
}

func TestSelectionPTYHelper(t *testing.T) {
	path := os.Getenv("AUROSCOPE_TEST_SELECTION_HELPER")
	if path == "" {
		return
	}
	client := paruClient{config: runConfig{paruPath: path, stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}}
	targets, err := client.selectPackages([]string{"hello"})
	if err != nil || !reflect.DeepEqual(targets, []string{"aur/hello"}) {
		t.Fatalf("selection=%q, %v", targets, err)
	}
	foreground, err := foregroundProcessGroup(os.Stdin.Fd())
	if err != nil || foreground != syscall.Getpgrp() {
		t.Fatalf("foreground=%d, %v", foreground, err)
	}
}

func TestSelectionTerminalCancellationDrainsCapture(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	t.Setenv("READY_FILE", ready)
	paru := writeExecutable(t, dir, "paru", `#!/bin/sh
trap 'exit 130' INT
: > "$READY_FILE"
while :; do sleep 1; done
`)
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	signals := make(chan os.Signal, 1)
	client := paruClient{config: runConfig{paruPath: paru, stdin: strings.NewReader(""), stdout: slave, stderr: slave, signals: signals}}
	done := make(chan error, 1)
	go func() { _, err := client.selectPackages([]string{"hello"}); done <- err }()
	waitForFile(t, ready)
	signals <- syscall.SIGINT
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "status 130") {
			t.Fatalf("cancel: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("selection capture did not terminate after SIGINT")
	}
}
