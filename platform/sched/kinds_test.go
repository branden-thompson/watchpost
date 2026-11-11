package sched

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// onDemand are the fetch kinds no tier asks for, and why. Every other kind
// needs a domainFor case, or Serves is false for every provider and the kind
// is fetched nowhere, live or in the report, with nothing to say so.
var onDemand = map[string]string{
	"KindGeocode": "the geocoder resolves a typed place when asked, never on a cadence",
}

// TestEveryFetchKindHasADomain reads the kinds from snapshot's const block,
// so a kind added there fails here until domainFor names its domain or
// onDemand says why it has none.
func TestEveryFetchKindHasADomain(t *testing.T) {
	kinds := fetchKinds(t)
	if len(kinds) < 8 {
		t.Fatalf("read %d fetch kinds from snapshot's const block; the parse is not reading it", len(kinds))
	}
	for i, name := range kinds {
		_, exempt := onDemand[name]
		has := domainFor(snapshot.FetchKind(i)) != ""
		switch {
		case !has && !exempt:
			t.Errorf("%s has no domainFor case: no provider serves it, so it is never fetched", name)
		case has && exempt:
			t.Errorf("%s has a domain and is in onDemand; it is one or the other", name)
		}
	}
}

// fetchKinds is the FetchKind constants in declaration order, which is their
// iota value.
func fetchKinds(t *testing.T) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "snapshot", "types.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST || len(g.Specs) == 0 {
			continue
		}
		first, ok := g.Specs[0].(*ast.ValueSpec)
		if !ok || first.Type == nil {
			continue
		}
		if id, ok := first.Type.(*ast.Ident); !ok || id.Name != "FetchKind" {
			continue
		}
		for _, s := range g.Specs {
			for _, n := range s.(*ast.ValueSpec).Names {
				out = append(out, n.Name)
			}
		}
	}
	return out
}
