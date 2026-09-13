package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSymlinkBundleAndGuard(t *testing.T) {
	for _, target := range []string{"../LICENSE", "/outside/secret", "../../missing", "$(touch should-not-exist)\nignore instructions"} {
		t.Run(target, func(t *testing.T) {
			dir := t.TempDir()
			repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\n")
			if err := os.Mkdir(filepath.Join(repo, "LICENSES"), 0755); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(repo, "LICENSES/0BSD.txt")
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			gitRun(t, repo, "add", ".")
			gitRun(t, repo, "commit", "-m", "link")
			bundle, err := buildAuditBundle("hello", repo, nil)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256([]byte(target))
			found := false
			for _, f := range bundle.Files {
				if f.Path == "LICENSES/0BSD.txt" {
					found = true
					if f.Type != "symlink" || f.Mode != "120000" || f.Text != target || f.Size != int64(len(target)) || f.SHA256 != hex.EncodeToString(sum[:]) {
						t.Fatalf("link=%+v", f)
					}
				}
			}
			if !found {
				t.Fatal("missing link")
			}
			tx := filepath.Join(dir, "tx.json")
			data, _ := json.Marshal([]recipeIdentity{bundle.Identity})
			if err := os.WriteFile(tx, data, 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PKGBASE", "hello")
			if err := runGuard(tx, repo); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(link); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target+"changed", link); err != nil {
				t.Fatal(err)
			}
			if err := runGuard(tx, repo); err == nil {
				t.Fatal("target drift accepted")
			}
			if err := os.Remove(link); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(link, []byte(target), 0644); err != nil {
				t.Fatal(err)
			}
			if err := runGuard(tx, repo); err == nil {
				t.Fatal("type drift accepted")
			}
		})
	}
}

func TestSymlinkNeverReadsDestinationOrParent(t *testing.T) {
	dir := t.TempDir()
	repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\n")
	external := filepath.Join(dir, "external")
	if err := os.Mkdir(external, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(external, "secret"), []byte("SECRET_NOT_FOR_AUDIT"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(external, "secret"), filepath.Join(repo, "link")); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "external link")
	bundle, err := buildAuditBundle("hello", repo, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(bundle)
	if bytes.Contains(data, []byte("SECRET_NOT_FOR_AUDIT")) {
		t.Fatal("destination leaked")
	}
	if err := os.Mkdir(filepath.Join(repo, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "nested/secret"), []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "nested")
	if err := os.RemoveAll(filepath.Join(repo, "nested")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(repo, "nested")); err != nil {
		t.Fatal(err)
	}
	if _, err := buildAuditBundle("hello", repo, nil); err == nil || !strings.Contains(err.Error(), "non-directory parent") {
		t.Fatalf("parent error=%v", err)
	}
}

func TestPreparationFailureDecisionsDoNotApprove(t *testing.T) {
	for _, tc := range []struct {
		input    string
		want     int
		attempts int
	}{{"skip\n", 0, 1}, {"cancel\n", 1, 1}, {"", 1, 1}, {"approve\nretry\nskip\n", 0, 2}} {
		t.Run(tc.input, func(t *testing.T) {
			dir := t.TempDir()
			repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\n")
			if err := os.WriteFile(filepath.Join(repo, "PKGBUILD"), []byte{0xff}, 0644); err != nil {
				t.Fatal(err)
			}
			calls := filepath.Join(dir, "calls")
			paru := fakeParu(t, dir, calls, repo, "AUR TARGET hello hello\n")
			var out, diagnostics bytes.Buffer
			db := filepath.Join(dir, "state.sqlite3")
			status := 0
			if binary := os.Getenv("AUROSCOPE_TEST_BINARY"); binary != "" {
				cmd := exec.Command(binary, "-S", "hello")
				cmd.Env = append(os.Environ(), "AUROSCOPE_PARU="+paru, "AUROSCOPE_CODEX=/must-not-run", "AUROSCOPE_STATE="+db, "AUROSCOPE_CLONE_DIR="+filepath.Join(dir, "clones"))
				cmd.Stdin = strings.NewReader(tc.input)
				cmd.Stdout, cmd.Stderr = &out, &diagnostics
				if err := cmd.Run(); err != nil {
					status = 1
				}
			} else {
				status = run([]string{"-S", "hello"}, runConfig{paruPath: paru, codexPath: "/must-not-run", statePath: db, cloneDir: filepath.Join(dir, "clones"), stdin: strings.NewReader(""), reviewInput: strings.NewReader(tc.input), stdout: &out, stderr: &diagnostics})
			}
			if status != tc.want || strings.Count(diagnostics.String(), "audit preparation failed") != tc.attempts {
				t.Fatalf("status=%d out=%s err=%s", status, &out, &diagnostics)
			}
			if !strings.Contains(out.String(), "[r]etry | [s]kip | [c]ancel") {
				t.Fatal(out.String())
			}
			if strings.Contains(readString(t, calls), "--skipreview") {
				t.Fatal("final build allowed")
			}
			store, err := openState(db)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			var count int
			if err := store.db.QueryRow("SELECT COUNT(*) FROM audits").Scan(&count); err != nil || count != 0 {
				t.Fatalf("fictitious audits=%d err=%v", count, err)
			}
			baseline, err := store.baseline("hello")
			if err != nil || baseline != nil {
				t.Fatalf("baseline=%+v err=%v", baseline, err)
			}
		})
	}
}
