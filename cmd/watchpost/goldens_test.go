package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// updateGoldens rewrites the golden gate's baseline from the tree.
var updateGoldens = flag.Bool("update-goldens", false, "rewrite 06_docs/testinv/goldens.tsv from the tree (B0's close only)")

// goldenTrees are the trees 0.19.6 splits, whose testdata the gate holds.
var goldenTrees = []string{"modes/tty", "app"}

// goldenBaseline is the gate's baseline, a hash and a path a line, captured
// at B0's closing commit; goldenLists are the batches' named lists (0.19.6 A-33),
// one path a line, each the files that batch re-records or adds.
const (
	goldenBaseline = "06_docs/testinv/goldens.tsv"
	goldenLists    = "06_docs/testinv/goldens-*.txt"
)

// TestNoGoldenChangesOutsideABatchsList is 0.19.6 B0.5's golden gate (RX-10,
// A-12, A-29, A-33): every file under the two trees' testdata either matches
// a file of B0's closing commit by content, so a pure rename passes, or is
// named in a batch's list. A re-recorded golden, or one added, outside every
// list fails: a golden that changes outside a listed re-recording is a
// regression, never a ruling candidate (A-29). Seen failing on a planted
// re-capture and a planted addition.
func TestNoGoldenChangesOutsideABatchsList(t *testing.T) {
	now := goldenHashes(t, "../..")
	if *updateGoldens {
		var b strings.Builder
		b.WriteString("# the golden gate's baseline (0.19.6 B0.5): every file under modes/tty's and app's testdata at B0's close, sha256 and path\n")
		for _, p := range slices.Sorted(maps.Keys(now)) { // bounded by the files (P10-02)
			b.WriteString(now[p] + "\t" + p + "\n")
		}
		if err := os.WriteFile(filepath.Join("../..", goldenBaseline), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	base, listed := goldenRecord(t)
	if len(base) < 100 {
		t.Fatalf("COULD NOT RUN: the baseline holds %d hashes; the trees' testdata held 108 files at B0's close", len(base))
	}
	planted := map[string]string{"modes/tty/testdata/a.golden": "re-captured", "app/testdata/new.golden": "added", "app/testdata/moved.golden": firstHash(base)}
	if got := goldenProblems(planted, base, nil); len(got) != 2 {
		t.Fatalf("COULD NOT RUN: a planted re-capture, addition and rename were judged %q; want the first two refused", got)
	}
	for _, p := range goldenProblems(now, base, listed) { // bounded by the findings (P10-02)
		t.Error(p)
	}
}

// goldenProblems is every file whose content matches no baseline file and
// whose path no batch's list names.
func goldenProblems(now map[string]string, base, listed map[string]bool) []string {
	var out []string
	for _, p := range slices.Sorted(maps.Keys(now)) { // bounded by the files (P10-02)
		if !base[now[p]] && !listed[p] {
			out = append(out, p+" differs from every file of B0's closing commit and is on no batch's list (06_docs/testinv/goldens-<batch>.txt): re-recorded or added outside a listed re-recording")
		}
	}
	return out
}

// firstHash is one of the baseline's hashes, for a planted rename.
func firstHash(base map[string]bool) string {
	return slices.Sorted(maps.Keys(base))[0]
}

// goldenRecord reads the baseline's hashes and every batch list's paths.
func goldenRecord(t *testing.T) (base, listed map[string]bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("../..", goldenBaseline))
	if err != nil {
		t.Fatalf("%v: capture it at B0's close with -update-goldens", err)
	}
	base = map[string]bool{}
	for _, l := range strings.Split(string(raw), "\n") { // bounded by the baseline (P10-02)
		if h, _, ok := strings.Cut(l, "\t"); ok && !strings.HasPrefix(l, "#") {
			base[h] = true
		}
	}
	lists, err := filepath.Glob(filepath.Join("../..", goldenLists))
	if err != nil {
		t.Fatal(err)
	}
	listed = map[string]bool{}
	for _, list := range lists { // bounded by the batches (P10-02)
		raw, err := os.ReadFile(list)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range strings.Split(string(raw), "\n") { // bounded by the list (P10-02)
			if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
				listed[l] = true
			}
		}
	}
	return base, listed
}

// goldenHashes is every file under the two trees' testdata, by its path from
// the root, with its content's sha256.
func goldenHashes(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, tree := range goldenTrees { // bounded by the two trees (P10-02)
		err := filepath.WalkDir(filepath.Join(root, tree), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.Contains(filepath.ToSlash(p), "/testdata/") {
				return err
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(raw)
			rel, err := filepath.Rel(root, p)
			out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}
