package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsurePrivateDirDoesNotTraverseBuildArtifacts(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission regression must run as an unprivileged user")
	}
	root := filepath.Join(t.TempDir(), "clones")
	artifact := filepath.Join(root, "hermes-agent-desktop", "pkg")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(artifact, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(artifact, 0o755) })
	if _, err := os.ReadDir(artifact); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("test requires an unreadable build directory, got %v", err)
	}
	if err := ensurePrivateDir(root); err != nil {
		t.Fatalf("prepare clone directory with unreadable build artifacts: %v", err)
	}
	for path, want := range map[string]os.FileMode{root: 0o700, artifact: 0} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s permissions = %o, want %o", path, got, want)
		}
	}
}

func TestEnsurePrivateDirCreatesRootAndRejectsFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "parent", "clones")
	if err := ensurePrivateDir(root); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("clone root mode = %v", info.Mode())
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensurePrivateDir(file); err == nil {
		t.Fatal("accepted a regular file as clone directory")
	}
}
