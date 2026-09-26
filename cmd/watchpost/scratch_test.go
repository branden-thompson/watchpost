package main

// scratch_test.go — no scratch test reaches the tree (0.18.0 batch 16).
//
// A throwaway test that prints what a window draws is how a layout gets looked
// at while it is built; one - `zz_peek_test.go`, `TestZZPeek` - was committed
// with batch 16 and taken out after. So the gate asks the files on disk, not
// the index: a scratch file fails the run before it can be staged.

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoScratchTestIsInTheTree(t *testing.T) {
	root := filepath.Join("..", "..")
	seen := 0
	err := filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() && (e.Name() == ".git" || e.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if e.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		seen++
		if strings.HasPrefix(e.Name(), "zz_") {
			t.Errorf("%s is a scratch test: take it out before the gate", path)
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), "func TestZZ") && !strings.HasSuffix(path, "scratch_test.go") {
			t.Errorf("%s holds a scratch test (func TestZZ...): take it out before the gate", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen < 100 {
		t.Fatalf("%d test files seen; this check has lost its subject", seen)
	}
}
