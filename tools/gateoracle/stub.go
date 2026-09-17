package gateoracle

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// The stubs read these from the environment; the oracle sets them per run.
const (
	EnvLog    = "ORACLE_LOG"    // the record every invocation appends to
	EnvStatus = "ORACLE_STATUS" // the directory of painted statuses
	EnvRoot   = "ORACLE_ROOT"   // the tree, for resolving script paths
	EnvBin    = "ORACLE_BIN"    // the stub directory, to find the REAL shell past it
	EnvBuilt  = "ORACLE_BUILT"  // the list of stubs `go build -o` has written
)

// tools are the toolchain commands the oracle stubs on PATH. A command NOT here
// is real; adding a checker that is neither a script nor `go run` nor built by
// `go build -o` means adding it here.
func tools() []string {
	return []string{"go", "gofmt", "a2dh", "golangci-lint", "govulncheck", "shasum", "sha256sum"}
}

// interpreters answer for their `scripts/` argument — a script's own shebang
// brings it here through `env` — and refuse anything else, so `python3 -m x` is
// UNJUDGEABLE rather than unseen. The shells run the REAL shell for `-c`.
func interpreters() []string { return []string{"sh", "bash", "python3", "expect"} }

func shells() []string { return []string{"sh", "bash"} }

// osShipped is every stubbed name the OS itself provides under /usr/bin or
// /bin; their ABSENCE cannot be simulated by PATH, and the ceiling says so.
func osShipped() []string {
	var out []string
	for _, name := range append(tools(), interpreters()...) { // bounded by the stub list (P10-02)
		for _, dir := range []string{"/usr/bin", "/bin"} { // bounded by the two OS directories (P10-02)
			if isExecutable(filepath.Join(dir, name)) && !contains(out, name) {
				out = append(out, name)
			}
		}
	}
	return out
}

// StubMain is the stub binary's main: the role is argv[0]'s name — bin/go,
// bin/sh, a tool, or a binary `go build -o` wrote — and the result is the exit
// status.
func StubMain(argv []string) int {
	if len(argv) == 0 {
		return 3
	}
	role, args := filepath.Base(argv[0]), argv[1:]
	switch {
	case role == "go":
		return goStub(args)
	case contains(interpreters(), role):
		return interpreterStub(role, args)
	case contains(tools(), role):
		return answer(role, args)
	}
	if key, ok := builtKey(argv[0]); ok {
		return answer("built:"+key, args)
	}
	return unjudged(role, "is not a role the stub plays")
}

// unjudged RECORDS a refusal — `unjudged:<role>` — so a refusal under `|| true`
// is seen by the oracle rather than swallowed, and exits 3.
func unjudged(role, why string) int {
	if _, err := recordInvocation(os.Getenv(EnvLog), "unjudged:"+role); err != nil {
		fmt.Fprintf(os.Stderr, "oracle: cannot record the refusal of %s: %v\n", role, err)
	}
	fmt.Fprintf(os.Stderr, "oracle: %s %s — not judged\n", role, why)
	return 3
}

// answer records the invocation and exits with the status painted for it, else
// for its key, else 0. `--self-test` before `--` makes it a control.
func answer(key string, args []string) int {
	if isSelfTest(args) {
		key = "ctl:" + key
	}
	inv, err := recordInvocation(os.Getenv(EnvLog), key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle: cannot record %s: %v\n", key, err)
		return 3
	}
	for _, k := range []string{inv, key} { // the invocation's paint, then the key's (P10-02)
		if raw, err := os.ReadFile(filepath.Join(os.Getenv(EnvStatus), encodeKey(k))); err == nil {
			if code, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
				return code
			}
		}
	}
	return 0
}

// goStub answers by sub-command, honours treelock's `--` contract (`go run … --
// CMD` runs CMD once its own status is green), and when it "builds" writes a
// recording stub — a symlink to this binary — at the output path.
func goStub(args []string) int {
	cwd, _ := os.Getwd()
	root := os.Getenv(EnvRoot)
	key := goKey(args, moduleOf(root))
	if pkg, ok := strings.CutPrefix(key, "go:run:"); ok {
		key = "go:run:" + localPackage(pkg, cwd, root)
	}
	if code := answer(key, args); code != 0 {
		return code
	}
	if len(args) > 0 && args[0] == "run" {
		for i, a := range args { // bounded by the arguments (P10-02)
			if a == "--" && i+1 < len(args) {
				return execArgs(args[i+1:])
			}
		}
	}
	if out := buildOutput(args, cwd); out != "" {
		if err := writeBuiltStub(out); err != nil {
			fmt.Fprintf(os.Stderr, "oracle: cannot write the built stub %s: %v\n", out, err)
			return 3
		}
	}
	return 0
}

// interpreterStub answers for the script among its arguments. A shell asked for
// a command string (`-c` anywhere in its leading flags) runs the REAL shell, and
// the script inside the string records through its own shebang.
func interpreterStub(role string, args []string) int {
	if contains(shells(), role) && hasShellCommandFlag(args) {
		real := realCommand(role)
		if real == "" {
			return unjudged(role, "has no real shell past the stubs")
		}
		return execArgs(append([]string{real}, args...))
	}
	cwd, _ := os.Getwd()
	if s := scriptArg(args, cwd, os.Getenv(EnvRoot)); s != "" {
		return answer(s, args)
	}
	return unjudged(role, "run without a scripts/ file argument")
}

// ---- the decisions, as functions -----------------------------------------------------------

// goKey is `go:<sub>`, `go:mod:<verb>`, or `go:run:<package>`.
func goKey(args []string, module string) string {
	if len(args) == 0 {
		return "go:"
	}
	switch args[0] {
	case "mod":
		if len(args) > 1 {
			return "go:mod:" + args[1]
		}
	case "run":
		return "go:run:" + packageOf(args[1:], module)
	}
	return "go:" + args[0]
}

// packageOf is the first argument shaped like a package, before `--`. The
// value after `-o` is an output path, never a package, whatever it looks like.
func packageOf(args []string, module string) string {
	skip := false
	for _, a := range args { // bounded by the arguments (P10-02)
		switch {
		case a == "--":
			return ""
		case skip:
			skip = false
		case a == "-o":
			skip = true
		case strings.HasPrefix(a, "-"):
		default:
			if p, ok := packageShaped(a, module); ok {
				return p
			}
		}
	}
	return ""
}

// packageShaped recognises `.`, `./x`, `../x`, `/x`, `x.go` (its directory),
// `<module>/x` (as `./x`) and `host.tld/x`; `foo` — the value of `-tags foo` —
// is not a package.
func packageShaped(a, module string) (string, bool) {
	switch {
	case a == ".":
		return ".", true
	case strings.HasPrefix(a, "./") || strings.HasPrefix(a, "../") || strings.HasPrefix(a, "/"):
		return localClean(a), true
	case strings.HasSuffix(a, ".go"):
		return localClean("./" + a), true
	case module != "" && strings.HasPrefix(a, module+"/"):
		return localClean("./" + strings.TrimPrefix(a, module+"/")), true
	}
	if first, _, _ := strings.Cut(a, "/"); strings.Contains(first, ".") {
		return a, true
	}
	return "", false
}

// localPackage resolves a relative package (`.`, `./x`, `../y`) against the
// working directory and names it from the tree root, so `cd tools/x && go run
// .` is `./tools/x`. A package that is not relative, or lies outside the tree,
// is returned as it is (absolute if outside).
func localPackage(pkg, cwd, root string) string {
	if pkg != "." && !strings.HasPrefix(pkg, "./") && !strings.HasPrefix(pkg, "../") {
		return pkg
	}
	abs := filepath.Clean(filepath.Join(cwd, pkg))
	rel, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return abs
	}
	if rel == "." {
		return "."
	}
	return "./" + filepath.ToSlash(rel)
}

// localClean cleans a local package path, takes a file's directory, and keeps
// the `./` that marks it local.
func localClean(p string) string {
	if strings.HasSuffix(p, ".go") {
		p = path.Dir(p)
	}
	p = path.Clean(p)
	if p == "." || strings.HasPrefix(p, "../") || strings.HasPrefix(p, "/") {
		return p
	}
	return "./" + p
}

// buildOutput is where `go build -o` or `go test -c -o` will write, resolved for
// the directory form; "" when nothing is written.
func buildOutput(args []string, cwd string) string {
	if len(args) == 0 {
		return ""
	}
	building := args[0] == "build" || (args[0] == "test" && contains(args, "-c"))
	if !building {
		return ""
	}
	out := ""
	for i, a := range args[1:] { // bounded by the arguments (P10-02)
		switch {
		case a == "-o" && i+2 < len(args):
			out = args[i+2]
		case strings.HasPrefix(a, "-o="):
			out = strings.TrimPrefix(a, "-o=")
		}
	}
	if out == "" {
		return ""
	}
	base := path.Base(packageOf(args[1:], ""))
	if args[0] == "test" {
		base += ".test" // what `go test -c` names a binary written into a directory
	}
	if strings.HasSuffix(out, "/") {
		return path.Join(out, base)
	}
	dir := out
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(cwd, out)
	}
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return path.Join(out, base)
	}
	return out
}

// hasShellCommandFlag is true when a `c` sits in the shell's leading flags —
// `-c`, `-ec`, `-e -o pipefail -c` — before the first operand.
func hasShellCommandFlag(args []string) bool {
	for i := 0; i < len(args); i++ { // bounded by the arguments (P10-02)
		a := args[i]
		switch {
		case a == "--":
			return false
		case strings.HasPrefix(a, "--"):
			continue
		case strings.HasPrefix(a, "-") || strings.HasPrefix(a, "+"):
			if strings.Contains(a[1:], "c") {
				return true
			}
			if strings.HasSuffix(a, "o") { // `-o pipefail`: the option's value follows
				i++
			}
		default:
			return false
		}
	}
	return false
}

// scriptArg is the first argument that is an existing file under root/scripts,
// as a root-relative path — resolved against cwd, so `cd scripts && ./x.sh` and
// `$(CURDIR)/scripts/x.sh` are one key. A command string is not a file.
func scriptArg(args []string, cwd, root string) string {
	scripts := filepath.Join(root, "scripts") + string(filepath.Separator)
	for _, a := range args { // bounded by the arguments (P10-02)
		if strings.HasPrefix(a, "-") {
			continue
		}
		p := a
		if !filepath.IsAbs(p) {
			p = filepath.Join(cwd, p)
		}
		p, err := filepath.EvalSymlinks(p)
		if err != nil {
			continue
		}
		info, err := os.Stat(p)
		if err != nil || info.IsDir() || !strings.HasPrefix(p, scripts) {
			continue
		}
		return filepath.ToSlash(strings.TrimPrefix(p, root+string(filepath.Separator)))
	}
	return ""
}

// isSelfTest is a `--self-test` or `-self-test` before `--`.
func isSelfTest(args []string) bool {
	for _, a := range args { // bounded by the arguments (P10-02)
		switch a {
		case "--":
			return false
		case "--self-test", "-self-test":
			return true
		}
	}
	return false
}

// moduleOf is go.mod's module path, or "".
func moduleOf(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") { // bounded by go.mod (P10-02)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// ---- process plumbing ----------------------------------------------------------------------

// execArgs replaces this process with argv, as a shell's exec would.
func execArgs(argv []string) int {
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle: %v\n", err)
		return 127
	}
	if err := syscall.Exec(bin, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "oracle: exec %s: %v\n", bin, err)
		return 126
	}
	return 0
}

// realCommand finds name on PATH past the stub directory.
func realCommand(name string) string {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) { // bounded by PATH (P10-02)
		if dir == os.Getenv(EnvBin) {
			continue
		}
		if p := filepath.Join(dir, name); isExecutable(p) {
			return p
		}
	}
	return ""
}

func isExecutable(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

// writeBuiltStub makes `out` a symlink to this binary and lists it, so a recipe
// that runs the "binary" reaches a stub keyed `built:<path>`.
func writeBuiltStub(out string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	_ = os.Remove(out)
	if err := os.Symlink(exe, out); err != nil {
		return err
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	return appendLine(os.Getenv(EnvBuilt), abs)
}

// builtKey is the tree-relative path of a built stub invoked as argv0, if it
// is one.
func builtKey(argv0 string) (string, bool) {
	abs, err := filepath.Abs(argv0)
	if err != nil {
		return "", false
	}
	raw, _ := os.ReadFile(os.Getenv(EnvBuilt))
	if !contains(strings.Split(string(raw), "\n"), abs) {
		return "", false
	}
	root := os.Getenv(EnvRoot)
	if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel), true
	}
	return abs, true
}
