package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexMinimumVersionAdmission(t *testing.T) {
	for _, tc := range []struct {
		banner string
		admit  bool
	}{
		{"codex-cli 0.150.1", true},
		{"codex-cli 0.150.2\n", true},
		{"codex-cli 0.151.0\r\n", true},
		{"codex-cli 0.153.4\n", true},
		{"codex-cli 0.1000.0", true},
		{"codex-cli 1.0.0", true},
		{"codex-cli 9.9.9", true},
		{"codex-cli 1000000000000000000000000000000.0.0", true},
		{"codex-cli 0.150.0", false},
		{"codex-cli 0.149.9999", false},
		{"codex-cli 0.9.9", false},
		{"codex-cli 0.0.0", false},
		{"codex-cli 0.153.4-alpha.1", false},
		{"codex-cli 0.153.4+build", false},
		{"codex-cli 00.153.4", false},
		{"codex-cli 0.0153.4", false},
		{"codex-cli 0.153.04", false},
		{"codex-cli +1.0.0", false},
		{"codex-cli -1.0.0", false},
		{"codex-cli 0.153", false},
		{"codex-cli 0.153.4.0", false},
		{"codex-cli v0.153.4", false},
		{"codex-cli ０.153.4", false},
		{"codex 0.153.4", false},
		{"codex-cli\t0.153.4", false},
		{"codex-cli  0.153.4", false},
		{" codex-cli 0.153.4", false},
		{"codex-cli 0.153.4 ", false},
		{"codex-cli 0.153.4\nwarning", false},
		{"codex-cli 0.153.4\ncodex-cli 0.153.4", false},
		{"codex-cli 0.153.4\n\n", false},
		{"codex-cli 0.153.4\r", false},
		{"", false},
	} {
		t.Run(tc.banner, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("CODEX_VERSION_BANNER", tc.banner)
			marker := filepath.Join(dir, "exec-started")
			t.Setenv("CODEX_EXEC_MARKER", marker)
			path := writeExecutable(t, dir, "codex", `#!/bin/sh
if test "$1" = "--version"; then
  printf '%s' "$CODEX_VERSION_BANNER"
  exit 0
fi
: > "$CODEX_EXEC_MARKER"
exit 42
`)
			client := codexClient{path: path}
			err := client.verifyVersion(runConfig{}.withDefaults())
			if tc.admit {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "minimum stable version is 0.150.1") {
				t.Fatalf("version error = %v", err)
			}
			if _, err := client.audit(auditBundle{}, runConfig{}); err == nil {
				t.Fatal("invalid version reached an audit")
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("exec ran for rejected version: %v", err)
			}
		})
	}
}
