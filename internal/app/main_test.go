package app

import (
	"os"
	"testing"
)

// Tests must not read a developer's actual provider selection or credentials.
// Individual configuration tests replace this directory with their own fixture.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "auroscope-test-config-*")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_CONFIG_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
