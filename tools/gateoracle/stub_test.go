package gateoracle

import (
	"os"
	"path/filepath"
	"testing"
)

// THE STUBS' PURE FUNCTIONS, EACH WITH A KNOWN CASE FIRST (FR-11.6). Rounds six
// through eight found the shell versions of these wrong: `-c` honoured only as
// `$1`, `-o=` unparsed, `tr / _` not injective, `-tags foo` taken as the package.

func TestGoKey(t *testing.T) {
	cases := []struct {
		name string
		args []string
		key  string
	}{
		{"test", []string{"test", "-race", "-count=1", "./..."}, "go:test"},
		{"mod tidy", []string{"mod", "tidy", "-diff"}, "go:mod:tidy"},
		{"run a local tool", []string{"run", "./tools/authoring", "-self-test"}, "go:run:./tools/authoring"},
		{"run with -tags before the package", []string{"run", "-tags", "foo", "./tools/x"}, "go:run:./tools/x"},
		{"run with -tags=foo", []string{"run", "-tags=foo", "./tools/x"}, "go:run:./tools/x"},
		{"run a module-path package is local", []string{"run", "example.com/mod/tools/x"}, "go:run:./tools/x"},
		{"run the file form is its directory", []string{"run", "tools/x/main.go"}, "go:run:./tools/x"},
		{"run with a trailing slash is cleaned", []string{"run", "./tools/x/", "-self-test"}, "go:run:./tools/x"},
		{"run a third-party tool", []string{"run", "golang.org/x/vuln/cmd/govulncheck@latest", "./..."}, "go:run:golang.org/x/vuln/cmd/govulncheck@latest"},
		{"run the treelock delegate", []string{"run", "./tools/treelock", "-name", "verify", "--", "make", "verify-gates"}, "go:run:./tools/treelock"},
		{"run dot", []string{"run", "."}, "go:run:."},
		{"build", []string{"build", "-trimpath", "-o", "dist/x", "./cmd/watchpost"}, "go:build"},
		{"version", []string{"version"}, "go:version"},
		{"no sub-command", nil, "go:"},
	}
	for _, c := range cases { // bounded by the table (P10-02)
		if got := goKey(c.args, "example.com/mod"); got != c.key {
			t.Errorf("%s: goKey(%v) = %q, want %q", c.name, c.args, got, c.key)
		}
	}
}

func TestBuildOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"-o path", []string{"build", "-o", "dist/x", "./cmd/x"}, "dist/x"},
		{"-o=path", []string{"build", "-o=dist/x", "./cmd/x"}, "dist/x"},
		{"-o dir/ lands at the package basename", []string{"build", "-o", "out/", "./tools/lintb"}, "out/lintb"},
		{"-o an existing directory", []string{"build", "-o", "out", "./tools/lintb"}, "out/lintb"},
		{"go test -c is a build", []string{"test", "-c", "-o", "out/x.test", "./tools/x"}, "out/x.test"},
		{"go test without -c builds nothing", []string{"test", "-o", "out/x.test", "./tools/x"}, ""},
		{"build without -o is not handled", []string{"build", "./cmd/x"}, ""},
		{"run builds nothing", []string{"run", "-o", "x", "./cmd/x"}, ""},
	}
	for _, c := range cases { // bounded by the table (P10-02)
		if got := buildOutput(c.args, dir); got != c.want {
			t.Errorf("%s: buildOutput(%v) = %q, want %q", c.name, c.args, got, c.want)
		}
	}
}

func TestHasShellCommandFlag(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"-c", "./scripts/x.sh"}, true},
		{[]string{"-ec", "./scripts/x.sh || true"}, true},
		{[]string{"-e", "-o", "pipefail", "-c", "x"}, true},
		{[]string{"-eo", "pipefail", "-c", "x"}, true},
		{[]string{"./scripts/x.sh", "-c"}, false}, // after the script it is the script's flag
		{[]string{"-e", "./scripts/x.sh"}, false},
		{[]string{"--norc", "-c", "x"}, true},
		{nil, false},
	}
	for _, c := range cases { // bounded by the table (P10-02)
		if got := hasShellCommandFlag(c.args); got != c.want {
			t.Errorf("hasShellCommandFlag(%v) = %v, want %v", c.args, got, c.want)
		}
	}
}

func TestScriptArg(t *testing.T) {
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	for _, p := range []string{"scripts/lint-a.sh", "scripts/quality/x.py"} { // bounded by the fixture (P10-02)
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, p), []byte("#!/usr/bin/env sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name string
		args []string
		cwd  string
		want string
	}{
		{"relative from the root", []string{"./scripts/lint-a.sh", "--self-test"}, root, "scripts/lint-a.sh"},
		{"bare from the root", []string{"scripts/lint-a.sh"}, root, "scripts/lint-a.sh"},
		{"absolute", []string{filepath.Join(root, "scripts/lint-a.sh")}, root, "scripts/lint-a.sh"},
		{"from inside scripts/", []string{"./lint-a.sh"}, filepath.Join(root, "scripts"), "scripts/lint-a.sh"},
		{"an interpreter flag first", []string{"-u", "scripts/quality/x.py", "arg"}, root, "scripts/quality/x.py"},
		{"a command string is not a file", []string{"./scripts/lint-a.sh || echo see lint-a_test.sh"}, root, ""},
		{"a file that does not exist", []string{"./scripts/nope.sh"}, root, ""},
		{"outside scripts/", []string{"./go.mod"}, root, ""},
		{"no argument", nil, root, ""},
	}
	for _, c := range cases { // bounded by the table (P10-02)
		if got := scriptArg(c.args, c.cwd, root); got != c.want {
			t.Errorf("%s: scriptArg(%v) = %q, want %q", c.name, c.args, got, c.want)
		}
	}
}

func TestIsSelfTest(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"--self-test"}, true},
		{[]string{"run", "./tools/x", "-self-test"}, true},
		{[]string{"run", "./tools/treelock", "--", "make", "x", "-self-test"}, false}, // after -- belongs to the wrapped command
		{[]string{"--", "--self-test"}, false},
		{[]string{"check"}, false},
	}
	for _, c := range cases { // bounded by the table (P10-02)
		if got := isSelfTest(c.args); got != c.want {
			t.Errorf("isSelfTest(%v) = %v, want %v", c.args, got, c.want)
		}
	}
}

func TestEncodeKeyIsReversible(t *testing.T) {
	keys := []string{"go:test#2", "scripts/quality/lint.sh#1", "scripts/quality_lint.sh#1", "ctl:scripts/x.sh#1", "built:out/x#1", "a%b"}
	seen := map[string]string{}
	for _, k := range keys { // bounded by the table (P10-02)
		enc := encodeKey(k)
		if filepath.Base(enc) != enc {
			t.Errorf("encodeKey(%q) = %q contains a separator", k, enc)
		}
		if other, dup := seen[enc]; dup {
			t.Errorf("encodeKey collides: %q and %q both encode to %q", k, other, enc)
		}
		seen[enc] = k
		if back := decodeKey(enc); back != k {
			t.Errorf("decodeKey(encodeKey(%q)) = %q", k, back)
		}
	}
}

func TestInvocationKeyAndCounts(t *testing.T) {
	invs := []string{"go:test#1", "go:test#2", "scripts/x.sh#1", "ctl:scripts/x.sh#1"}
	if got := keyOf("go:test#2"); got != "go:test" {
		t.Errorf("keyOf = %q", got)
	}
	if got := keyOf("plain"); got != "plain" {
		t.Errorf("keyOf(plain) = %q", got)
	}
	counts := countsOf(invs)
	if counts["go:test"] != 2 || counts["scripts/x.sh"] != 1 || len(counts) != 3 {
		t.Errorf("countsOf = %v", counts)
	}
	if got := keysOf(invs); len(got) != 3 || got[0] != "go:test" {
		t.Errorf("keysOf = %v", got)
	}
}

func TestSpecialTarget(t *testing.T) {
	for name, want := range map[string]bool{".PHONY": true, ".DEFAULT": true, ".SUFFIXES": true, ".ONESHELL": true,
		".x-run": false, "lint-a": false, ".c.o": false} { // bounded by the table (P10-02)
		if got := specialTarget(name); got != want {
			t.Errorf("specialTarget(%q) = %v, want %v", name, got, want)
		}
	}
}
