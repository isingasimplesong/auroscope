package app

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestDriftSkipKeepsOtherTargets(t *testing.T) {
	dir := t.TempDir()
	repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\n")
	other := createRecipeRepo(t, dir, "other", "pkgname=other\n")
	calls := filepath.Join(dir, "calls")
	t.Setenv("CALLS", calls)
	t.Setenv("RECIPE_REPO", repo)
	t.Setenv("OTHER_REPO", other)
	t.Setenv("ATTEMPTED", filepath.Join(dir, "attempted"))
	// The real guard/subprocess handoff is covered by TestRecipeDriftRecovery.
	// Here, inject its result to focus on the mixed, multi-package target list.
	paru := writeExecutable(t, dir, "paru", `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$CALLS"
if [ "$1" = -P ]; then
  printf 'REPO TARGET extra tree\nAUR TARGET hello hello\nAUR TARGET other other\n'
  exit 0
fi
if [ "$1" = -G ]; then
  if [ "$2" = hello ]; then cp -R "$RECIPE_REPO" "$PWD/$2"
  else cp -R "$OTHER_REPO" "$PWD/$2"; fi
  exit 0
fi
if [ ! -f "$ATTEMPTED" ]; then
  printf '{"pkgbase":"hello"}' > "${PARU_CONF%/*}/approved.json.drift.json"
  touch "$ATTEMPTED"
  exit 1
fi
cp "${PARU_CONF%/*}/approved.json" "$APPROVAL_COPY"
test "$*" = '-S --skipreview -- tree other'
`)
	approvalCopy := filepath.Join(dir, "approved-copy.json")
	t.Setenv("APPROVAL_COPY", approvalCopy)
	config := runConfig{
		paruPath:  paru,
		codexPath: fakeCodex(t, dir, filepath.Join(dir, "bundle.json"), `{"summary":"reviewed","risk":"low","findings":[],"uncertainty":"","inspect":[]}`),
		statePath: filepath.Join(dir, "state.sqlite3"), cloneDir: filepath.Join(dir, "clones"),
		stdin: strings.NewReader(""), reviewInput: strings.NewReader("a\na\ns\n"),
		stdout: io.Discard, stderr: io.Discard,
	}
	if status := run([]string{"-S", "tree", "hello", "other"}, config); status != 0 {
		t.Fatalf("status = %d; calls = %s", status, readString(t, calls))
	}
	var approved []recipeIdentity
	readJSON(t, approvalCopy, &approved)
	if len(approved) != 1 || approved[0].Pkgbase != "other" {
		t.Fatalf("skip retained wrong approvals: %#v", approved)
	}
	// If fresh Paru resolution still requires hello as a dependency, its guard
	// must reject it rather than treating skip as permission to build.
	t.Setenv("PKGBASE", "hello")
	if err := runGuard(approvalCopy, repo); err == nil {
		t.Fatal("skipped dependency passed the guard")
	}
}
