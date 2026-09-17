package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOptionOperandsAreNotTargets(t *testing.T) {
	for _, args := range [][]string{
		{"-S", "--ignore", "linux", "hello"},
		{"-S", "--config", "/tmp/pacman.conf", "hello"},
		{"-S", "--ignore=linux", "hello"},
		{"-S", "--config=/tmp/pacman.conf", "hello"},
		{"-S", "--config", "relative.conf", "hello"},
		{"-S", "--ignore", "linux", "--ignore", "hello", "hello"},
	} {
		t.Run(args[1]+args[2], func(t *testing.T) {
			targets, err := initialTargets(args, paruClient{})
			if err != nil || !reflect.DeepEqual(targets, []string{"hello"}) {
				t.Fatalf("initialTargets(%q) = %q, %v", args, targets, err)
			}
		})
	}
}

func TestParuArgumentBoundaries(t *testing.T) {
	for _, tc := range []struct {
		args, options, targets []string
	}{
		{[]string{"-S", "--ignore", "hello", "hello"}, []string{"-S", "--ignore", "hello"}, []string{"hello"}},
		{[]string{"-S", "hello", "--config", "./relative path.conf", "--ignore=linux"}, []string{"-S", "--config", "./relative path.conf", "--ignore=linux"}, []string{"hello"}},
		{[]string{"-S", "--ignore", "--", "hello"}, []string{"-S", "--ignore", "--"}, []string{"hello"}},
		{[]string{"-S", "--", "hello", "-Sup", "--repo", "--build"}, []string{"-S"}, []string{"hello", "-Sup", "--repo", "--build"}},
		{[]string{"-S", "--rebuild", "hello"}, []string{"-S", "--rebuild"}, []string{"hello"}},
		{[]string{"-S", "--rebuild=all", "hello"}, []string{"-S", "--rebuild=all"}, []string{"hello"}},
		{[]string{"-Sb/tmp/db", "-r", "/tmp/root", "hello"}, []string{"-Sb/tmp/db", "-r", "/tmp/root"}, []string{"hello"}},
		{[]string{"-S", "--config", "-Sup", "hello"}, []string{"-S", "--config", "-Sup"}, []string{"hello"}},
		{[]string{"-S", "--ignore", "--repo", "hello"}, []string{"-S", "--ignore", "--repo"}, []string{"hello"}},
		{[]string{"-S", "--config", "--sysupgrade", "hello"}, []string{"-S", "--config", "--sysupgrade"}, []string{"hello"}},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			parsed, err := parseParuArguments(tc.args)
			if err != nil || !reflect.DeepEqual(parsed.options, tc.options) || !reflect.DeepEqual(parsed.targets, tc.targets) {
				t.Fatalf("parse = %#v, %v", parsed, err)
			}
			if !commandMayBuildAUR(tc.args) || isSystemUpgrade(tc.args) {
				t.Fatal("operand/target text changed dispatch")
			}
			targets, err := initialTargets(tc.args, paruClient{})
			if err != nil || !reflect.DeepEqual(targets, tc.targets) {
				t.Fatalf("targets = %q, %v", targets, err)
			}
			got, err := auditedInstallArgs(tc.args, []string{"kept"}, "--skipreview")
			want := append(append([]string(nil), tc.options...), "--skipreview", "--", "kept")
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("handoff = %q, %v; want %q", got, err, want)
			}
		})
	}
}

// Runs unchanged against the installed artifact in the supported-Paru gate.
func TestOptionOperandsHandoff(t *testing.T) {
	for _, options := range [][]string{
		{"--ignore", "linux"}, {"--ignore=linux"},
		{"--config", "/tmp/pacman.conf"}, {"--config=/tmp/pacman.conf"},
		{"--config", "./relative path.conf"}, {"--config=relative.conf"},
		{"--ignore", "linux", "--ignore", "hello", "--config", "hello"},
		{"--ignore", "--"}, {"--config", "-Sup"}, {"--ignore", "--repo"},
	} {
		for _, decision := range []string{"approve", "skip"} {
			t.Run(strings.Join(options, "_")+"/"+decision, func(t *testing.T) {
				dir := t.TempDir()
				repo := createRecipeRepo(t, dir, "hello", "pkgname=hello\npkgver=1\n")
				calls := filepath.Join(dir, "calls")
				fake := fakeParu(t, dir, calls, repo, "REPO TARGET extra tree\nAUR TARGET hello hello\n")
				t.Setenv("OPTION_FAKE_PARU", fake)
				argvFile := filepath.Join(dir, "argv")
				t.Setenv("OPTION_ARGV_FILE", argvFile)
				paru := writeExecutable(t, dir, "record-paru", `#!/bin/sh
printf 'CALL\n' >> "$OPTION_ARGV_FILE"
printf '%s\n' "$@" >> "$OPTION_ARGV_FILE"
exec "$OPTION_FAKE_PARU" "$@"
`)
				codex := fakeCodex(t, dir, filepath.Join(dir, "bundle"), `{"summary":"checked","risk":"low","findings":[],"uncertainty":"","inspect":[]}`)
				args := append([]string{"-S"}, options...)
				args = append(args, "--", "tree", "hello")
				var output bytes.Buffer
				config := runConfig{paruPath: paru, codexPath: codex,
					statePath: filepath.Join(dir, "state"), cloneDir: filepath.Join(dir, "clones"),
					stdin: strings.NewReader(decision + "\n"), stdout: &output, stderr: &output}
				status := runOptionFixture(t, args, config)
				if status != 0 {
					t.Fatalf("status=%d: %s", status, &output)
				}
				final := append([]string{"-S"}, options...)
				if decision == "approve" {
					final = append(final, "--skipreview", "--", "tree", "hello")
				} else {
					final = append(final, "--", "tree")
				}
				want := "CALL\n-P\n--order\n--\ntree\nhello\nCALL\n-G\nhello\nCALL\n" + strings.Join(final, "\n") + "\n"
				if got := readString(t, argvFile); got != want {
					t.Fatalf("argv=%q; want %q", got, want)
				}
			})
		}
	}
}

func TestOptionOperandsRejectInvalidBeforeParu(t *testing.T) {
	for _, args := range [][]string{
		{"-S", "--config"}, {"-S", "--ignore"}, {"-Sb"},
		{"-S", "--unknown-option", "hello"}, {"-S", "--unknown-option=hello"},
		{"-SZ", "hello"}, {"-S", "--needed=yes", "hello"},
		{"-S", "--config", "relative.conf", "./PKGBUILD"},
		{"-S", "--ignore", "linux", "--", "hello", "../recipe"},
		{"-S", "--", "hello", "-pkg", "/tmp/recipe"},
		{"-B", "."}, {"--build", "/tmp/recipe"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "called")
			paru := writeExecutable(t, dir, "paru", "#!/bin/sh\ntouch "+posixShellQuote(marker)+"\nexit 99\n")
			var output bytes.Buffer
			config := runConfig{paruPath: paru, codexPath: paru,
				statePath: filepath.Join(dir, "state"), cloneDir: filepath.Join(dir, "clones"),
				stdin: strings.NewReader(""), stdout: &output, stderr: &output}
			if status := runOptionFixture(t, args, config); status != statusFailure {
				t.Fatalf("status=%d: %s", status, &output)
			}
			for _, path := range []string{marker, config.statePath, config.cloneDir} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("unexpected side effect %s: %v", path, err)
				}
			}
			if output.Len() == 0 {
				t.Fatal("missing actionable diagnostic")
			}
		})
	}
}

func runOptionFixture(t *testing.T, args []string, config runConfig) int {
	t.Helper()
	if binary := os.Getenv("AUROSCOPE_TEST_BINARY"); binary != "" {
		cmd := exec.Command(binary, args...)
		cmd.Env = append(os.Environ(), "AUROSCOPE_PARU="+config.paruPath,
			"AUROSCOPE_CODEX="+config.codexPath, "AUROSCOPE_STATE="+config.statePath,
			"AUROSCOPE_CLONE_DIR="+config.cloneDir, "XDG_CONFIG_HOME="+t.TempDir())
		cmd.Stdin, cmd.Stdout, cmd.Stderr = config.stdin, config.stdout, config.stderr
		return commandStatus(cmd.Run())
	}
	return run(args, config)
}

func TestOptionOperandsSeparatorReachesOrder(t *testing.T) {
	dir := t.TempDir()
	calls := filepath.Join(dir, "calls")
	t.Setenv("OPTION_ORDER_CALLS", calls)
	paru := writeExecutable(t, dir, "paru", `#!/bin/sh
printf '%s\n' "$@" > "$OPTION_ORDER_CALLS"
printf 'MISSING -Sup\n'
exit 1
`)
	var output bytes.Buffer
	status := runOptionFixture(t, []string{"-S", "--", "hello", "-Sup"}, runConfig{
		paruPath: paru, codexPath: "/must-not-run", statePath: filepath.Join(dir, "state"),
		cloneDir: filepath.Join(dir, "clones"), stdin: strings.NewReader(""), stdout: &output, stderr: &output,
	})
	if status != statusFailure || !strings.Contains(output.String(), "MISSING") {
		t.Fatalf("expected native missing-target resolution: %d %s", status, &output)
	}
	if got := readString(t, calls); got != "-P\n--order\n--\nhello\n-Sup\n" {
		t.Fatalf("targets lost separator in planning: %q", got)
	}
}

func TestOptionOperandsNativePassThrough(t *testing.T) {
	for _, args := range [][]string{
		{"-S", "--repo", "--config", "/tmp/pacman.conf", "hello"},
		{"-Syu", "--print-format", "%n %v", "--config", "relative.conf"},
		{"-Q", "--config", "-Syu"},
	} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			dir := t.TempDir()
			var output bytes.Buffer
			paru := writeExecutable(t, dir, "paru", "#!/bin/sh\nprintf '%s\\n' \"$@\"\nexit 17\n")
			status := runOptionFixture(t, args, runConfig{paruPath: paru, stdin: strings.NewReader(""), stdout: &output, stderr: &output})
			if status != 17 || output.String() != strings.Join(args, "\n")+"\n" {
				t.Fatalf("status=%d: %q", status, &output)
			}
		})
	}
}
