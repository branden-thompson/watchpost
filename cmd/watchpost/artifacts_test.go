package main

// artifacts_test.go — no built artefact is tracked in git.
//
// THE DENYLIST IS NOT THE GUARD. `.gitignore` names the tool binaries anyone has
// thought of; it cannot name a tool that does not exist yet, and two binaries
// reached commits while it stood — the second written by an author who had read
// the paragraph describing the first. A list of names fails exactly when a new
// tool arrives, which is the only time it is asked anything.
//
// SO THIS ASKS THE INDEX. Every tracked file is checked for an executable magic
// number, and anything large enough to be a build product has to be declared.
// `go build ./tools/<x>/` drops its output in the working directory, `git add -A`
// sweeps it in, and neither step says anything — the defect has no symptom until
// a clone is 5 MB heavier and the history cannot be rewritten without a rule.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// execMagic is the first bytes of the executable formats a build here can emit.
//
// THE THREE PLATFORMS THIS PROJECT BUILDS FOR, and a fourth form: Mach-O is
// little-endian on arm64 and big-endian on the older universal binaries, and
// both spellings are the same mistake.
var execMagic = [][]byte{
	{0x7f, 'E', 'L', 'F'},    // ELF — Linux
	{0xcf, 0xfa, 0xed, 0xfe}, // Mach-O 64-bit little-endian — macOS arm64/amd64
	{0xce, 0xfa, 0xed, 0xfe}, // Mach-O 32-bit little-endian
	{0xca, 0xfe, 0xba, 0xbe}, // Mach-O universal / fat
	{'M', 'Z'},               // PE — Windows
}

// largeAllowed are the tracked files over sizeFloor that are MEANT to be there.
//
// EVERY ROW IS A THING A READER DOWNLOADS ON PURPOSE: the documentation images
// and the embedded place-name table. A new row is a deliberate act, which is the
// point — the cost of adding one is naming it here, in front of a reviewer.
var largeAllowed = map[string]string{
	"domains/locations/geodata/data/cities_trim.tsv.gz": "the embedded place-name index; the app cannot resolve a location without it",
	"domains/locations/geodata/data/zips_trim.tsv.gz":   "the embedded postcode table, the other half of the offline resolver",
}

// sizeFloor is where a tracked file stops being source and starts being a
// payload. Documentation images sit above it legitimately and are matched by
// extension rather than listed one by one, so adding a screenshot needs no edit.
const sizeFloor = 400_000

func TestNoTrackedBinaries(t *testing.T) {
	root := filepath.Join("..", "..")
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v — this check measures nothing without it", err)
	}
	paths := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	if len(paths) < 100 {
		t.Fatalf("the index holds %d files; this check has lost its subject", len(paths))
	}

	for _, p := range paths { // bounded by the index (P10-02)
		if p == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(root, p))
		if err != nil || info.IsDir() {
			continue // a file in the index and not on disk is a different problem
		}
		if isExecutable(t, filepath.Join(root, p)) {
			t.Errorf("%s is a compiled executable and it is TRACKED.\n"+
				"Build output belongs in ./dist/ and never in the index: `go build ./tools/<x>/` "+
				"drops it here and `git add -A` sweeps it in silently.\n"+
				"Remove it with `git rm --cached %s`, delete the working copy, and add /%s to .gitignore.", p, p, p)
			continue
		}
		if info.Size() <= sizeFloor || isDocAsset(p) {
			continue
		}
		if _, ok := largeAllowed[p]; !ok {
			t.Errorf("%s is %d bytes and nothing declares it.\n"+
				"A tracked file this large is a payload rather than source. If it belongs, add it to "+
				"largeAllowed with the reason a reader would want it; if it is build output, it belongs in ./dist/.",
				p, info.Size())
		}
	}
}

// isDocAsset reports whether a path is a documentation image.
//
// MATCHED BY SHAPE, NOT BY NAME, so adding a screenshot to the README needs no
// edit here. They are large on purpose and a reader fetches them deliberately.
func isDocAsset(p string) bool {
	if !strings.HasPrefix(p, "docs/img/") && !strings.Contains(p, "/06_docs/") && !strings.HasPrefix(p, "06_docs/") {
		return false
	}
	switch strings.ToLower(filepath.Ext(p)) {
	case ".png", ".gif", ".jpg", ".jpeg", ".svg", ".webp":
		return true
	}
	return false
}

// isExecutable reads the file's first bytes and compares them against the
// formats a Go build emits. The mode bit is not the test: scripts are executable
// and belong, and a binary committed without the bit is the same 5 MB.
func isExecutable(t *testing.T, path string) bool {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, 4)
	n, _ := f.Read(head)
	head = head[:n]
	for _, m := range execMagic { // bounded by the format list (P10-02)
		if len(head) >= len(m) && string(head[:len(m)]) == string(m) {
			return true
		}
	}
	return false
}
