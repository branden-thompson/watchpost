package main

import (
	"maps"
	"slices"
	"strings"
)

// deckOnlyTests are the test files design AP says move intact into
// app/internal/deck, each held to reaching the deck's code alone (0.19.6
// design AP, RX's second batch-0 check; A-45): F-158's test (AP-7.1). The app
// trace (AP-7.2, app/deck_trace_test.go) joins the list in the deck's prep
// commit, where its hooks come from FillConfig rather than livePipelines.
var deckOnlyTests = []string{"app/cutback_hazard_test.go"}

// movesWithTheDeck are the planned packages a deck-only test may reach: the
// deck itself, and app's test support, a package the deck's tests import.
var movesWithTheDeck = []string{"deck", "apptest"}

// outsideTheDeck is every object of a non-test app file that the test file
// reaches, through the test helpers it calls, transitively, whose planned
// package is not one the deck's tests can reach: each a name that would have
// to be exported, or the test re-written, at the deck's move. A test file
// with no declaration indexed is reported as itself.
func (ix *index) outsideTheDeck(pl *planned, file string) []string {
	byFunc := map[string][]use{} // a test file's top-level function, by file and name -> its uses
	for _, u := range ix.uses {  // bounded by the uses (P10-02)
		if u.test && strings.HasPrefix(u.file, "app/") {
			byFunc[u.file+"\t"+u.encl] = append(byFunc[u.file+"\t"+u.encl], u)
		}
	}
	var stack []string
	for k := range byFunc { // bounded by the test functions (P10-02)
		if strings.HasPrefix(k, file+"\t") {
			stack = append(stack, k)
		}
	}
	if len(stack) == 0 {
		return []string{file + " declares nothing the index read: the file is missing or not in app"}
	}
	seen := map[string]bool{}
	out := map[string]bool{}
	for len(stack) > 0 { // bounded by app's test functions: each is visited once (P10-02)
		k := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[k] {
			continue
		}
		seen[k] = true
		for _, u := range byFunc[k] { // bounded by the function's body (P10-02)
			switch {
			case !strings.HasPrefix(u.decl, "app/"):
			case strings.HasSuffix(u.decl, "_test.go"):
				stack = append(stack, u.decl+"\t"+ix.symName(u.obj))
			default:
				to := pl.file[u.decl]
				if m, ok := pl.method[ix.symName(u.obj)]; ok {
					to = m
				}
				if !slices.Contains(movesWithTheDeck, to) {
					out[ix.symName(u.obj)+" ("+u.decl+", planned "+to+")"] = true
				}
			}
		}
	}
	return slices.Sorted(maps.Keys(out))
}
