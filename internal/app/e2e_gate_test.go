package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the actual drift scenario assertion, not a duplicate predicate.
// Only the bounded shell block runs; sudo is a fake function, so this test
// never starts Docker, a package manager, AURoscope, or any package recipe.
func TestSupportedParuDriftGate(t *testing.T) {
	data, err := os.ReadFile("../../scripts/e2e-supported-paru.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	start := strings.Index(script, "# A committed mutation")
	end := strings.Index(script, "# Skipping the only AUR target")
	if start < 0 || end <= start {
		t.Fatal("cannot locate the bounded drift scenario")
	}
	// Ground the positive control in the production guard's actual diagnostic,
	// including its observed and approved commit/manifest identities.
	fixture := t.TempDir()
	repo := createRecipeRepo(t, fixture, "hello", "pkgname=hello\n")
	identity, _, err := readRecipeIdentity("hello", repo)
	if err != nil {
		t.Fatal(err)
	}
	approved, err := json.Marshal([]recipeIdentity{identity})
	if err != nil {
		t.Fatal(err)
	}
	tx := filepath.Join(fixture, "approved.json")
	if err := os.WriteFile(tx, approved, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PKGBASE", "hello")
	if err := runGuard(tx, repo); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "PKGBUILD"), []byte("pkgname=hello\n# drift\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = runGuard(tx, repo)
	if err == nil {
		t.Fatal("positive control did not produce a guard refusal")
	}
	diagnostic := "auroscope guard: " + err.Error()
	for _, tc := range []struct {
		name       string
		status     int
		newHandoff bool
		diagnostic string
		pass       bool
	}{
		{"network before handoff", 1, false, "network unavailable", false},
		{"acquisition before handoff", 1, false, "acquire recipe failed", false},
		{"audit before handoff", 1, false, "audit failed", false},
		{"resolution after handoff", 1, true, "resolution failed", false},
		{"diagnostic without new handoff", 1, false, diagnostic, false},
		{"wrong package", 1, true, strings.Replace(diagnostic, "hello:", "other:", 1), false},
		{"truncated diagnostic", 1, true, "auroscope guard: recipe identity drift for hello", false},
		{"near match", 1, true, diagnostic + "-other", false},
		{"zero status despite evidence", 0, true, diagnostic, false},
		{"actual guard refusal", 1, true, diagnostic, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			calls := filepath.Join(dir, "paru-calls")
			output := filepath.Join(dir, "drift-output")
			// Earlier successful scenarios must not satisfy this one's proof.
			if err := os.WriteFile(calls, []byte("-S --skipreview -- hello\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(output, []byte(diagnostic+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			block := strings.NewReplacer("/tmp/paru-calls", calls, "/tmp/drift-output", output).Replace(script[start:end])
			handoff := ""
			if tc.newHandoff {
				handoff = "printf '%s\\n' '-S --noconfirm --rebuild --skipreview -- hello' >> " + posixShellQuote(calls)
			}
			body := fmt.Sprintf("#!/bin/bash\nset -euo pipefail\nsudo() {\n%s\nprintf '%%s\\n' %s >&2\nreturn %d\n}\n%s", handoff, posixShellQuote(tc.diagnostic), tc.status, block)
			path := filepath.Join(dir, "scenario.sh")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := exec.Command("bash", path).CombinedOutput()
			if (err == nil) != tc.pass {
				t.Fatalf("gate pass=%v, want %v; error=%v output=%s", err == nil, tc.pass, err, got)
			}
		})
	}
}
