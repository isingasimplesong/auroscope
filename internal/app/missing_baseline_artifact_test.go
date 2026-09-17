package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Runs unchanged on the installed artifact in disposable Arch. Only fixture
// executables run: recipes remain inert and no host package manager is used.
func TestMissingBaselineReview(t *testing.T) {
	binary := os.Getenv("AUROSCOPE_TEST_BINARY")
	if binary == "" {
		binary = filepath.Join(t.TempDir(), "auroscope")
		if out, err := exec.Command("go", "build", "-o", binary, "../../cmd/auroscope").CombinedOutput(); err != nil {
			t.Fatalf("build: %v\n%s", err, out)
		}
	}
	for _, tc := range []struct {
		name, input, failure   string
		status, finals, builds int
	}{
		{"approve", "inspect\napprove\n", "", 0, 1, 1},
		{"cancel", "cancel\n", "", 1, 0, 0},
		{"eof", "", "", 1, 0, 0},
		{"skip", "skip\n", "", 0, 0, 0},
		{"invalid-audit", "approve\ncancel\n", "audit", 1, 0, 0},
		{"paru-failure", "approve\n", "paru", 37, 1, 0},
		{"identity-drift", "approve\ncancel\n", "drift", 1, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\npkgver=1\n")
			clones := filepath.Join(dir, "clones")
			if err := os.Mkdir(clones, 0o700); err != nil {
				t.Fatal(err)
			}
			local := filepath.Join(clones, "hello")
			gitRun(t, dir, "clone", repo, local)
			if err := os.WriteFile(filepath.Join(local, "PKGBUILD"), []byte("pkgname=hello\npkgver=1\npkgrel=2\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := snapshotEditedWorktree(local, map[string]bool{}); err != nil {
				t.Fatal(err)
			}
			old, _, err := readRecipeIdentity("hello", local)
			if err != nil {
				t.Fatal(err)
			}
			statePath := filepath.Join(dir, "state.sqlite3")
			store, err := openState(statePath)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if err := store.advanceBaselines([]recipeIdentity{old}); err != nil {
				t.Fatal(err)
			}
			if err := store.recordAudit("hello", old, "", auditReport{Summary: "previous edit", Risk: "low"}, "approve"); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(local); err != nil {
				t.Fatal(err)
			}
			calls := filepath.Join(dir, "calls")
			builds := filepath.Join(dir, "builds")
			codexCalls := filepath.Join(dir, "codex-calls")
			t.Setenv("CODEX_CALLS", codexCalls)
			bundlePath := filepath.Join(dir, "bundle.json")
			report := `{"summary":"current recipe reviewed","risk":"low","findings":[],"uncertainty":"","inspect":[]}`
			if tc.failure == "audit" {
				report = `{}`
			}
			codex := fakeCodex(t, dir, bundlePath, report)
			paru := writeExecutable(t, dir, "paru", `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$CALLS"
case "$1" in
-P) printf 'AUR TARGET hello hello\n'; exit 0 ;;
-G) git clone --quiet "$RECIPE_REPO" "$PWD/hello"; exit 0 ;;
esac
[ "$1" = -S ]
if [ "$FAILURE" = paru ]; then exit 37; fi
cd "$AUROSCOPE_CLONE_DIR/hello"
if [ "$FAILURE" = drift ]; then printf '\n# drift\n' >> PKGBUILD; fi
hook=''
while IFS= read -r line; do
  case "$line" in 'PreBuildCommand = '*) hook=${line#PreBuildCommand = } ;; esac
done < "$PARU_CONF"
[ -n "$hook" ]
PKGBASE=hello sh -c "$hook"
printf 'build\n' >> "$BUILDS"
`)
			cmd := exec.Command(binary, "-S", "hello")
			cmd.Stdin = strings.NewReader(tc.input)
			cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+filepath.Join(dir, "config"),
				"AUROSCOPE_PARU="+paru, "AUROSCOPE_CODEX="+codex,
				"AUROSCOPE_STATE="+statePath, "AUROSCOPE_CLONE_DIR="+clones,
				"RECIPE_REPO="+repo, "CALLS="+calls, "BUILDS="+builds, "FAILURE="+tc.failure)
			output, err := cmd.CombinedOutput()
			if got := commandStatus(err); got != tc.status {
				t.Fatalf("status %d, want %d\n%s", got, tc.status, output)
			}
			if !strings.Contains(string(output), "Historical comparison unavailable") {
				t.Fatalf("missing user warning:\n%s", output)
			}
			if tc.name == "approve" && !strings.Contains(string(output), "Previous successful commit: "+old.Commit) {
				t.Fatal("inspection lost previous commit")
			}
			if tc.failure == "drift" && !strings.Contains(string(output), "recipe identity drift") {
				t.Fatalf("guard not reached:\n%s", output)
			}
			if got := strings.Count(readString(t, codexCalls), "audit\n"); got != 1 {
				t.Fatalf("audits=%d", got)
			}
			if got := strings.Count(readString(t, calls), "--skipreview"); got != tc.finals {
				t.Fatalf("finals=%d", got)
			}
			built, _ := os.ReadFile(builds)
			if got := strings.Count(string(built), "build\n"); got != tc.builds {
				t.Fatalf("builds=%d", got)
			}
			var bundle auditBundle
			readJSON(t, bundlePath, &bundle)
			if bundle.Mode != "full" || bundle.PreviousCommit != old.Commit || bundle.Warning == "" || bundle.Diff != "" {
				t.Fatalf("bundle=%#v", bundle)
			}
			_, files, err := readRecipeIdentity("hello", repo)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(bundle.Files, files) {
				t.Fatal("provider did not receive full current recipe")
			}
			baseline, err := store.baseline("hello")
			if err != nil {
				t.Fatal(err)
			}
			want := old
			if tc.builds == 1 {
				want = bundle.Identity
			}
			if baseline.Commit != want.Commit || baseline.ManifestDigest != want.ManifestDigest {
				t.Fatal("baseline advanced without successful approved build")
			}
			var historical int
			if err := store.db.QueryRow(`SELECT count(*) FROM audits WHERE "commit" = ? AND decision = 'approve'`, old.Commit).Scan(&historical); err != nil {
				t.Fatal(err)
			}
			if historical != 1 {
				t.Fatal("lost old audit")
			}
			if tc.builds == 1 {
				var previous string
				if err := store.db.QueryRow(`SELECT previous_commit FROM audits ORDER BY id DESC LIMIT 1`).Scan(&previous); err != nil {
					t.Fatal(err)
				}
				if previous != old.Commit {
					t.Fatal("successful audit lost historical reference")
				}
			}
		})
	}
}
