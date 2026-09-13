package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The same regression can exercise the installed package in disposable Arch.
// Paru and Codex are deterministic fixtures; package scripts are only data.
func TestAuditIncludesAuxiliaryScripts(t *testing.T) {
	dir := t.TempDir()
	repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\npkgver=1\ninstall=hello.install\nprepare() { sh helper; }\n")
	marker := filepath.Join(dir, "recipe-executed")
	contents := map[string]string{
		"hello.install": "post_install() { sh helper; }\npost_remove() { sh cleanup; }\n",
		"helper":        "#!/bin/sh\nprintf executed > '" + marker + "'\n",
		"cleanup":       "#!/bin/sh\nprintf removed > '" + marker + "'\n",
		"hello.service": "[Service]\nExecStart=/usr/bin/hello\n",
	}
	for path, text := range contents {
		if err := os.WriteFile(filepath.Join(repo, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Waypaper-style tracked link: include target text, never dereference.
	if err := os.Symlink("helper", filepath.Join(repo, "helper-link")); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "auxiliary recipe files")
	previous := gitTrim(t, repo, "rev-parse", "HEAD")
	// Build leftovers are not recipe context, even if they look like scripts.
	if err := os.WriteFile(filepath.Join(repo, "build-leftover.sh"), []byte("exit 99\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bundlePath := filepath.Join(dir, "bundle.json")
	config := runConfig{
		paruPath:  fakeParu(t, dir, filepath.Join(dir, "calls"), repo, "AUR TARGET hello hello\n"),
		codexPath: fakeCodex(t, dir, bundlePath, `{"summary":"install script reviewed","risk":"medium","findings":[{"file":"hello.install","line":2,"evidence":"post_remove","explanation":"removal hook"}],"uncertainty":"","inspect":["helper","cleanup","hello.service"]}`),
		statePath: filepath.Join(dir, "state.sqlite3"),
		cloneDir:  filepath.Join(dir, "clones"),
	}
	for _, mode := range []string{"full", "diff", "unchanged"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "diff" {
				if err := os.WriteFile(filepath.Join(repo, "PKGBUILD"), []byte("pkgname=hello\npkgver=1\npkgrel=2\ninstall=hello.install\nprepare() { sh helper; }\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				gitRun(t, repo, "add", "PKGBUILD")
				gitRun(t, repo, "commit", "-m", "update caller only")
			}
			var output bytes.Buffer
			if binary := os.Getenv("AUROSCOPE_TEST_BINARY"); binary != "" {
				cmd := exec.Command(binary, "-S", "hello")
				cmd.Env = append(os.Environ(),
					"AUROSCOPE_PARU="+config.paruPath,
					"AUROSCOPE_CODEX="+config.codexPath,
					"AUROSCOPE_STATE="+config.statePath,
					"AUROSCOPE_CLONE_DIR="+config.cloneDir)
				cmd.Stdin = strings.NewReader("approve\n")
				cmd.Stdout, cmd.Stderr = &output, &output
				if err := cmd.Run(); err != nil {
					t.Fatalf("installed audit: %v\n%s", err, output.String())
				}
			} else {
				config.stdin = strings.NewReader("")
				config.reviewInput = strings.NewReader("approve\n")
				config.stdout, config.stderr = &output, &output
				if status := run([]string{"-S", "hello"}, config); status != 0 {
					t.Fatalf("audit status %d\n%s", status, output.String())
				}
			}
			var bundle auditBundle
			readJSON(t, bundlePath, &bundle)
			if bundle.Mode != mode {
				t.Fatalf("mode = %q, want %q", bundle.Mode, mode)
			}
			if mode == "diff" && (bundle.PreviousCommit != previous || !strings.Contains(bundle.Diff, "pkgrel=2")) {
				t.Fatalf("lost baseline diff: %#v", bundle)
			}
			_, files, err := readRecipeIdentity("hello", repo)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(bundle.Files, files) {
				t.Fatalf("Codex did not receive the complete current recipe: %#v", bundle.Files)
			}
			for _, file := range bundle.Files {
				if file.Path == "build-leftover.sh" {
					t.Fatal("Codex received an untracked build leftover")
				}
			}
			for path, text := range contents {
				found := false
				for _, file := range bundle.Files {
					if file.Path == path && file.Text == text {
						found = true
					}
				}
				if !found {
					t.Errorf("Codex missing full content for %s", path)
				}
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("package content must not execute; marker stat = %v", err)
			}
		})
	}
}
