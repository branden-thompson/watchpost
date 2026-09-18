package gateoracle

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FixtureSource lays a specimen out as a small committed git repository — a
// Makefile, the required list, a go.mod and one Go file, the named scripts with
// env shebangs, a tag, and a git-ignored AGENTS.md as a developer machine has —
// under root/source, as the SOURCE the oracle clones from, so specimens go
// through the same CI-shaping as the real tree.
func FixtureSource(t Reporter, root, makefile, required string, scripts []string, extra map[string]string) string {
	t.Helper()
	source := filepath.Join(root, "source")
	files := map[string]string{
		"Makefile": makefile, "06_docs/required-gates.txt": required,
		"go.mod": "module fixture\n", "main.go": "package main\n\nfunc main() {}\n",
		".gitignore": "AGENTS.md\n", "AGENTS.md": "ignored on the developer machine\n",
	}
	// A SHEBANG AND NOTHING ELSE: the script never runs — its shebang brings the
	// interpreter stub, which answers for it — and a Go file carries no program
	// in shell (AP-SHELL-01).
	for _, sc := range scripts { // bounded by the script list (P10-02)
		files[sc] = "#!/usr/bin/env sh\n"
	}
	for name, body := range extra { // bounded by the extra-file map (P10-02)
		files[name] = body
	}
	for name, body := range files { // bounded by the file map (P10-02)
		path := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("COULD NOT RUN — %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatalf("COULD NOT RUN — %v", err)
		}
	}
	for _, args := range [][]string{
		{"init", "-q"}, {"add", "-A"},
		{"-c", "user.name=oracle", "-c", "user.email=oracle", "commit", "-q", "--no-verify", "-m", "fixture"},
		{"tag", "v0.0.0"},
	} { // bounded by the command list (P10-02)
		cmd := exec.Command("git", args...)
		cmd.Dir = source
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("COULD NOT RUN — git %v: %v\n%s", args, err, out)
		}
	}
	return source
}

// SpecimenOracle builds the oracle over a fixture, cloned as CI would have it;
// the returned cleanup removes everything.
func SpecimenOracle(t Reporter, makefile, required string, scripts []string, extra map[string]string) (*Oracle, func()) {
	t.Helper()
	root, err := os.MkdirTemp("", "gate-oracle-*")
	if err != nil {
		t.Fatalf("COULD NOT RUN — %v", err)
	}
	source := FixtureSource(t, root, makefile, required, scripts, extra)
	o := New(t, CloneForOracle(t, source, filepath.Join(root, "oracle")))
	return o, func() { _ = os.RemoveAll(root) }
}

// ---- the base fixture every specimen table mutates ----------------------------------

// BaseMakefile is a small Makefile that is CORRECT under every gate.
const BaseMakefile = `
DIST := dist
BINARY := fixture
.PHONY: verify verify-gates race test-tags lint-a lint-b gate-controls alloc-budget build-check mutant-anchors mutant-check release-matrix install-test
verify:
	@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates

verify-gates: race test-tags lint-a lint-b gate-controls alloc-budget build-check mutant-anchors mutant-check

race:
	go test -race -count=1 ./...

test-tags:
	go test -tags watchpost_debug -count=1 ./app

test:
	go test ./...

lint-a:
	@./scripts/lint-a.sh

lint-b:
	@scripts/lint-b.sh

mutant-anchors:
	@./scripts/quality/mutant-anchors.sh

gate-controls:
	@./scripts/lint-a.sh --self-test
	@./scripts/lint-b.sh --self-test
	@./scripts/quality/mutant-anchors.sh --self-test
	# a real comment beside a live control is fine
	@./scripts/quality/p10-unmatched_test.sh

alloc-budget:
	go test -count=1 -run 'AllocBudget' ./...

build-check:
	go build $(TRIMPATH) -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY) ./cmd/watchpost

mutant-check:
	@go test -tags mutants -count=1 ./06_docs/mutants/

release-matrix:
	go build $(TRIMPATH) -o dist/x-linux ./cmd/watchpost

install-test: release-matrix
	@./scripts/install-test.sh

quality-bench:
	go test ./x -run '^$$' -bench . -count 10

journey: build
	@expect scripts/quality/validate-journey.expect
`

// BaseRequired is the required list that matches BaseMakefile.
const BaseRequired = `
race
test-tags
lint-a
lint-b
gate-controls
alloc-budget
build-check
mutant-anchors
mutant-check
release-matrix
install-test
`

// AnchorMissing is what Sub leaves in a Makefile when its anchor was not found,
// so the specimen runner refuses it by name instead of judging the wrong text.
const AnchorMissing = "# ORACLE-ANCHOR-MISSING: "

// Sub is a Makefile mutation: old replaced by new, exactly once. A missing
// anchor is reported through AnchorMissing rather than by aborting here.
func Sub(old, new string) func(string) string {
	return func(s string) string {
		if !strings.Contains(s, old) {
			return s + "\n" + AnchorMissing + strings.ReplaceAll(old, "\n", "\\n") + "\n"
		}
		return strings.Replace(s, old, new, 1)
	}
}

// MissingAnchor is the anchor a mutated Makefile says it could not find, or "".
func MissingAnchor(mutated string) string {
	if i := strings.Index(mutated, AnchorMissing); i >= 0 {
		rest := mutated[i+len(AnchorMissing):]
		anchor, _, _ := strings.Cut(rest, "\n")
		return anchor
	}
	return ""
}

func Same(s string) string { return s }
