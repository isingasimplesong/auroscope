package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The same scenarios run against the installed package in the Arch gate.
func TestEmptyAURUpdate(t *testing.T) {
	for _, tc := range []struct {
		name, queryOutput               string
		queryStatus, nativeStatus, want int
		wantNative                      bool
	}{
		{"no updates", "", 1, 0, 0, true},
		{"empty successful query", "", 0, 0, 0, true},
		{"native error", "", 1, 42, 42, true},
		{"network error remains failure", "", 1, 1, 1, true},
		{"query failure", "", 42, 0, 42, false},
		{"query interrupted", "", 130, 0, 130, false},
		{"failed partial query", "hello\n", 1, 0, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			calls := filepath.Join(dir, "calls")
			confCopy := filepath.Join(dir, "native.conf")
			t.Setenv("CALLS", calls)
			t.Setenv("CONF_COPY", confCopy)
			t.Setenv("QUERY_OUTPUT", tc.queryOutput)
			t.Setenv("QUERY_STATUS", fmt.Sprint(tc.queryStatus))
			t.Setenv("NATIVE_STATUS", fmt.Sprint(tc.nativeStatus))
			paru := writeExecutable(t, dir, "paru", `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$CALLS"
case "$*" in
  '-Syu --repo') printf 'official phase\n'; exit 0 ;;
  '-Qua --quiet') printf '%s' "$QUERY_OUTPUT"; exit "$QUERY_STATUS" ;;
  '-Su --mode=aur --skipreview')
    test -f "$PARU_CONF"
    cp "$PARU_CONF" "$CONF_COPY"
    cp "$(dirname "$PARU_CONF")/approved.json" "$CONF_COPY.approved"
    printf '\033[1m:: packages not in the AUR: local-only\033[0m\n' >&2
    printf ':: marked out of date: old-bin\n' >&2
    printf 'native prompt: '
    read -r answer
    test "$answer" = yes
    printf ' there is nothing to do\n'
    exit "$NATIVE_STATUS" ;;
  *) exit 99 ;;
esac
`)
			configPath := filepath.Join(dir, "config", "auroscope", "config.json")
			t.Setenv("XDG_CONFIG_HOME", filepath.Dir(filepath.Dir(configPath)))
			t.Setenv("AUROSCOPE_PARU", paru)
			t.Setenv("AUROSCOPE_CODEX", "/does/not/exist")
			t.Setenv("AUROSCOPE_CLONE_DIR", filepath.Join(dir, "clones"))
			t.Setenv("AUROSCOPE_STATE", filepath.Join(dir, "state.sqlite3"))
			var stdout, stderr bytes.Buffer
			inputPath := filepath.Join(dir, "input")
			if err := os.WriteFile(inputPath, []byte("yes\n"), 0600); err != nil {
				t.Fatal(err)
			}
			input, err := os.Open(inputPath)
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			var status int
			if binary := os.Getenv("AUROSCOPE_TEST_BINARY"); binary != "" {
				cmd := exec.Command(binary)
				cmd.Stdin, cmd.Stdout, cmd.Stderr = input, &stdout, &stderr
				status = commandStatus(cmd.Run())
			} else {
				status = run(nil, runConfig{userConfig: true, stdin: input, stdout: &stdout, stderr: &stderr})
			}
			if status != tc.want {
				t.Fatalf("status = %d, want %d; stderr: %s", status, tc.want, &stderr)
			}
			if strings.Contains(readString(t, calls), "-Su --mode=aur --skipreview") != tc.wantNative {
				t.Fatalf("wrong native handoff: %s", readString(t, calls))
			}
			if tc.wantNative {
				if stdout.String() != "official phase\nnative prompt:  there is nothing to do\n" || stderr.String() != "\033[1m:: packages not in the AUR: local-only\033[0m\n:: marked out of date: old-bin\n" {
					t.Fatalf("native streams changed: stdout=%q stderr=%q", &stdout, &stderr)
				}
				var approved []recipeIdentity
				if err := json.Unmarshal([]byte(readString(t, confCopy+".approved")), &approved); err != nil || len(approved) != 0 {
					t.Fatalf("empty update approved recipes: %v, %v", approved, err)
				}
				t.Setenv("PKGBASE", "newly-discovered")
				if err := runGuard(confCopy+".approved", dir); err == nil || !strings.Contains(err.Error(), "unexpected pkgbase") {
					t.Fatalf("new recipe was not refused: %v", err)
				}
				conf := readString(t, confCopy)
				if !strings.Contains(conf, "PreBuildCommand = ") || !strings.Contains(conf, " __guard ") {
					t.Fatalf("missing pre-build guard: %s", conf)
				}
			}
			if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				t.Fatalf("empty update created provider configuration: %v", err)
			}
		})
	}
}

// A private libalpm database and local RPC fixture exercise real pinned Paru.
// No host package database, external AUR data, or package recipe is involved.
func TestEmptyUpdateRealParu(t *testing.T) {
	if os.Getenv("AUROSCOPE_E2E_INSIDE") != "1" || os.Getenv("AUROSCOPE_TEST_REAL_PARU") == "" {
		t.Skip("requires disposable Arch and pinned Paru")
	}
	dir := t.TempDir()
	db := filepath.Join(dir, "db")
	for _, name := range []string{"auroscope-test-missing", "auroscope-test-stale"} {
		pkgDir := filepath.Join(db, "local", name+"-1-1")
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}
		desc := fmt.Sprintf("%%NAME%%\n%s\n\n%%VERSION%%\n1-1\n\n%%BASE%%\n%s\n\n%%DESC%%\nPrivate test fixture\n\n%%ARCH%%\nx86_64\n\n", name, name)
		if err := os.WriteFile(filepath.Join(pkgDir, "desc"), []byte(desc), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(db, "local", "ALPM_DB_VERSION"), []byte("9\n"), 0644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"version":5,"type":"multiinfo","resultcount":1,"results":[{"ID":1,"Name":"auroscope-test-stale","PackageBaseID":1,"PackageBase":"auroscope-test-stale","Version":"1-1","Description":"Private test fixture","URL":"https://example.invalid","URLPath":"/snapshot.tar.gz","NumVotes":0,"Popularity":0,"OutOfDate":1,"Maintainer":"test","FirstSubmitted":1,"LastModified":1}]}`)
	}))
	defer server.Close()
	pacmanConf := filepath.Join(dir, "pacman.conf")
	if err := os.WriteFile(pacmanConf, []byte("[options]\nArchitecture = x86_64\n"), 0600); err != nil {
		t.Fatal(err)
	}
	paruConf := filepath.Join(dir, "paru.conf")
	if err := os.WriteFile(paruConf, []byte("[options]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PARU_CONF", paruConf)
	t.Setenv("TEST_DB", db)
	t.Setenv("TEST_PACMAN_CONF", pacmanConf)
	t.Setenv("TEST_RPC", server.URL)
	t.Setenv("LC_ALL", "C")
	paru := writeExecutable(t, dir, "paru", `#!/bin/sh
if [ "$*" = '-Syu --repo' ]; then
  printf 'official phase fixture\n'
  exit 0
fi
exec "$AUROSCOPE_TEST_REAL_PARU" --config "$TEST_PACMAN_CONF" --dbpath "$TEST_DB" --aurrpcurl "$TEST_RPC" --nodevel "$@"
`)
	query := exec.Command(paru, "-Qua", "--quiet")
	queryOut, err := query.Output()
	if commandStatus(err) != 1 || len(queryOut) != 0 {
		t.Fatalf("real empty query: status=%d stdout=%q error=%v", commandStatus(err), queryOut, err)
	}
	t.Setenv("AUROSCOPE_PARU", paru)
	t.Setenv("AUROSCOPE_CODEX", "/does/not/exist")
	t.Setenv("AUROSCOPE_CLONE_DIR", filepath.Join(dir, "clones"))
	t.Setenv("AUROSCOPE_STATE", filepath.Join(dir, "state.sqlite3"))
	var output bytes.Buffer
	var status int
	if binary := os.Getenv("AUROSCOPE_TEST_BINARY"); binary != "" {
		cmd := exec.Command(binary)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = strings.NewReader(""), &output, &output
		status = commandStatus(cmd.Run())
	} else {
		status = run(nil, runConfig{stdin: strings.NewReader(""), stdout: &output, stderr: &output})
	}
	if status != 0 {
		t.Fatalf("real empty update: status=%d output=%s", status, &output)
	}
	for _, text := range []string{"packages not in the AUR", "auroscope-test-missing", "marked out of date", "auroscope-test-stale", "there is nothing to do"} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("missing native %q in %s", text, &output)
		}
	}
}
