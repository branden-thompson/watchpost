package main

import (
	"strings"
	"testing"
)

// FULL TDD note: run() is the only behavior in B0's placeholder main; the cobra
// tree that replaces it at B1a arrives test-first.
func TestRunVersionFlag(t *testing.T) {
	if err := run([]string{"watchpost", "--version"}); err != nil {
		t.Fatalf("--version must succeed, got %v", err)
	}
}

func TestRunWithoutArgsStartsTheDashboard(t *testing.T) {
	// A bare `watchpost` is the dashboard — with no config yet it opens with
	// the Setup window (UAT 100), never a "run setup first" refusal. Headless
	// here, so the only acceptable failure is the terminal itself.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	err := run([]string{"watchpost"})
	if err != nil && strings.Contains(err.Error(), "watchpost setup") {
		t.Fatalf("no config must open the Setup window, not refuse: %v", err)
	}
}

func TestRunGuardsEmptyArgv(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("empty argv must fail the invariant, not panic")
	}
}
