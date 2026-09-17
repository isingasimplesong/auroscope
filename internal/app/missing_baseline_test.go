package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMissingBaselineAfterCloneCleanup(t *testing.T) {
	for _, source := range []string{"local-edit", "remote-history"} {
		t.Run(source, func(t *testing.T) {
			dir := t.TempDir()
			upstream := createRecipeRepo(t, dir, "hello", "pkgname=hello\npkgver=1\n")
			local := filepath.Join(dir, "local")
			gitRun(t, dir, "clone", upstream, local)
			if source == "remote-history" {
				appendRecipeCommit(t, upstream, "pkgrel=2\n")
				gitRun(t, local, "pull", "--ff-only")
			} else {
				if err := os.WriteFile(filepath.Join(local, "PKGBUILD"), []byte("pkgname=hello\npkgver=1\npkgrel=2\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := snapshotEditedWorktree(local, map[string]bool{}); err != nil {
					t.Fatal(err)
				}
			}
			id, _, err := readRecipeIdentity("hello", local)
			if err != nil {
				t.Fatal(err)
			}
			store, err := openState(filepath.Join(dir, "state.sqlite3"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if err := store.advanceBaselines([]recipeIdentity{id}); err != nil {
				t.Fatal(err)
			}
			if source == "remote-history" {
				gitRun(t, upstream, "reset", "--hard", "HEAD^")
				gitRun(t, upstream, "reflog", "expire", "--expire=now", "--all")
				gitRun(t, upstream, "gc", "--prune=now")
			}
			if err := os.RemoveAll(local); err != nil {
				t.Fatal(err)
			}
			gitRun(t, dir, "clone", upstream, local)
			previous, err := store.baseline("hello")
			if err != nil {
				t.Fatal(err)
			}
			bundle, err := buildAuditBundle("hello", local, previous)
			if err != nil {
				t.Fatalf("current recipe must remain auditable: %v", err)
			}
			if bundle.Mode != "full" || bundle.PreviousCommit != id.Commit || bundle.Diff != "" {
				t.Fatalf("lost historical context: %#v", bundle)
			}
			_, files, err := readRecipeIdentity("hello", local)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(bundle.Files, files) {
				t.Fatal("incomplete current recipe")
			}
			after, err := store.baseline("hello")
			if err != nil {
				t.Fatal(err)
			}
			if *after != *previous {
				t.Fatal("preparation advanced baseline")
			}
		})
	}
}

func TestBaselineGitErrorsRemainVisible(t *testing.T) {
	dir := t.TempDir()
	repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\n")
	previous := gitTrim(t, repo, "rev-parse", "HEAD")
	appendRecipeCommit(t, repo, "pkgrel=2\n")
	// A real corrupt historical object is not a missing history fallback.
	path := filepath.Join(repo, ".git", "objects", previous[:2], previous[2:])
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := buildAuditBundle("hello", repo, &packageBaseline{Commit: previous}); err == nil {
		t.Fatal("corrupt Git object silently hidden")
	}
	// An available commit with a missing tree must still fail at diff creation.
	repo = createRecipeRepo(t, dir, "other", "pkgname=other\n")
	previous = gitTrim(t, repo, "rev-parse", "HEAD")
	tree := gitTrim(t, repo, "rev-parse", "HEAD^{tree}")
	appendRecipeCommit(t, repo, "pkgrel=2\n")
	if err := os.Remove(filepath.Join(repo, ".git", "objects", tree[:2], tree[2:])); err != nil {
		t.Fatal(err)
	}
	if _, err := buildAuditBundle("other", repo, &packageBaseline{Commit: previous}); err == nil || !strings.Contains(err.Error(), "build recipe diff") {
		t.Fatalf("non-missing-commit diff failure hidden: %v", err)
	}
}
