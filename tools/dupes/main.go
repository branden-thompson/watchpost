// dupes — metric D: operations implemented more than once with no ratified reason.
//
// THE THIRD INSTRUMENT, AND THE FIRST ONE COMMITTED. DISCOVER recorded that the
// 0.14.1 duplicate-detection script "was never committed — the third instrument
// this project has used and thrown away, after sampler.sh and this phase's
// probes." A measurement that lives in someone's shell is a measurement the
// next release cannot repeat, which is why metric D has been deferred twice
// with a number nobody could reproduce. This one is in the tree, has a gate,
// and has a ledger.
//
// WHAT IT COUNTS. Two functions are the same OPERATION when their bodies have
// the same token STRUCTURE — the same sequence of token kinds, with every
// identifier and literal replaced by a placeholder. That catches the copy that
// was renamed, which is the shape this project keeps finding by hand: `hz` in
// severe.go while two other windows drew the same rule inline; one operation in
// four hand-written copies (issue #7).
//
// WHY A SIZE FLOOR, AND WHY THIS ONE. Without a floor this drowns in
// `func (x T) Y() string { return x.y }`, which is not duplication, it is Go.
// The floor counts AST NODES in the body, and 25 is CALIBRATED, not guessed:
// across this repository's 2,051 production functions the node count runs
// p10=10, p25=20, p50=41, p75=90, p99=351. A floor of 25 sits between the
// quartile of one-line accessors and the median real function, so it admits
// roughly the top 60% by size and excludes the trivia.
//
// The first version of this file used 40 and called it a token count. It is a
// node count, a substantial worked example measures 25, and the SELF-TEST
// caught that on its first run — the floor had been set by guess against a unit
// it was not measuring.
//
// usage: go run ./tools/dupes [-min 40] [-tests] [-ledger path] [-json]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type site struct {
	Func string `json:"func"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type group struct {
	Fingerprint string `json:"fingerprint"`
	Nodes       int    `json:"nodes"`
	Sites       []site `json:"sites"`
}

func main() {
	min := flag.Int("min", 25, "minimum body AST-node count to consider (calibrated; see the note above)")
	withTests := flag.Bool("tests", false, "include _test.go files")
	ledgerPath := flag.String("ledger", "06_docs/duplicates-ratified.md", "ratified-duplicate ledger")
	asJSON := flag.Bool("json", false, "emit JSON")
	selfTest := flag.Bool("self-test", false, "prove the detector fires on a known-bad fixture")
	flag.Parse()

	if *selfTest {
		os.Exit(runSelfTest(*min))
	}

	groups, err := scan(".", *min, *withTests)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dupes:", err)
		os.Exit(2)
	}
	ratified := readLedger(*ledgerPath)

	var unratified []group
	var drift []string
	for _, g := range groups {
		sites, known := ratified[g.Fingerprint]
		if !known {
			unratified = append(unratified, g)
			continue
		}
		if ok, why := covers(sites, g); !ok {
			unratified = append(unratified, g)
			drift = append(drift, g.Fingerprint+": "+why)
		}
	}
	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"groups": groups, "unratified": unratified, "min_nodes": *min,
		})
	} else {
		fmt.Printf("dupes: %d duplicate group(s) at >= %d nodes; %d ratified, %d NOT\n",
			len(groups), *min, len(groups)-len(unratified), len(unratified))
		for _, d := range drift {
			fmt.Printf("\n  RATIFIED GROUP CHANGED — %s\n", d)
		}
		for _, g := range unratified {
			fmt.Printf("\n  %s  (%d nodes, %d copies)\n", g.Fingerprint, g.Nodes, len(g.Sites))
			for _, s := range g.Sites {
				fmt.Printf("    %s:%d  %s\n", s.File, s.Line, s.Func)
			}
		}
	}
	if len(unratified) > 0 {
		fmt.Fprintf(os.Stderr, "\ndupes: %d duplicate group(s) carry no ratified reason.\n"+
			"Collapse them, or present them to the HUM LEAD and record the reason in %s.\n"+
			"A reason is RATIFIED, never self-issued (metric D hardening, project brief).\n",
			len(unratified), *ledgerPath)
		os.Exit(1)
	}
}

// fingerprintBody reduces a body to its token STRUCTURE: every identifier and
// literal becomes a placeholder, so a renamed copy fingerprints the same as its
// original. Comments are already excluded — the scanner does not emit them.
func fingerprintBody(fset *token.FileSet, src []byte, body *ast.BlockStmt) (string, int) {
	var b strings.Builder
	n := 0
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		switch x := node.(type) {
		case *ast.Ident:
			b.WriteString("i;")
		case *ast.BasicLit:
			b.WriteString("l;")
		default:
			fmt.Fprintf(&b, "%T;", x)
		}
		n++
		return true
	})
	return fmt.Sprintf("%x", hash(b.String())), n
}

func hash(s string) uint64 {
	var h uint64 = 1469598103934665603
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

func scan(root string, min int, withTests bool) ([]group, error) {
	fset := token.NewFileSet()
	byPrint := map[string][]site{}
	nodes := map[string]int{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			// third_party is vendored and not ours to collapse; 06_docs/mutants
			// holds deliberately mutated copies of real code, which is the one
			// place duplication is the POINT.
			if base == ".git" || base == "third_party" || base == "mutants" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !withTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			fp, n := fingerprintBody(fset, src, fn.Body)
			if n < min {
				continue
			}
			// THE RECEIVER'S TYPE, not "(recv)". A site name goes into a
			// ratified row and is compared against it, so it has to be the
			// name a person would write — "Opts.Distance", not a placeholder
			// that makes two different methods look identical.
			name := fn.Name.Name
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				t := fn.Recv.List[0].Type
				if star, ok := t.(*ast.StarExpr); ok {
					t = star.X
				}
				if id, ok := t.(*ast.Ident); ok {
					name = id.Name + "." + name
				} else if idx, ok := t.(*ast.IndexExpr); ok { // a generic receiver
					if id, ok := idx.X.(*ast.Ident); ok {
						name = id.Name + "." + name
					}
				}
			}
			byPrint[fp] = append(byPrint[fp], site{Func: name, File: path, Line: fset.Position(fn.Pos()).Line})
			nodes[fp] = n
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var out []group
	for fp, sites := range byPrint {
		if len(sites) < 2 {
			continue
		}
		sort.Slice(sites, func(i, j int) bool { return sites[i].File < sites[j].File })
		out = append(out, group{Fingerprint: fp[:12], Nodes: nodes[fp], Sites: sites})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nodes > out[j].Nodes })
	return out, nil
}

// readLedger takes ratified exemptions out of a markdown table:
//
//	| `fingerprint` | pkg.Func · pkg.Other | reason … RATIFIED … |
//
// A row without a RATIFIED marker does not count, on the same rule the P10
// ledger gate enforces — a reason nobody approved is not a reason.
//
// THE SITES ARE PART OF THE EXEMPTION, and leaving them out was a hole (red
// team, 2026-09-08). Keyed on the fingerprint alone, a row naming two specific
// functions granted "any set of functions with this AST shape, anywhere,
// forever": a THIRD copy joining the ratified group passed silently, and so did
// an unrelated function in another package that happened to share the shape.
// What the HUM LEAD ratified was two named functions, and that is what this
// records.
func readLedger(path string) map[string][]string {
	out := map[string][]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") || !strings.Contains(line, "RATIFIED") {
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cells) < 2 {
			continue
		}
		fp := strings.Trim(strings.TrimSpace(cells[0]), "`")
		if fp == "" {
			continue
		}
		var sites []string
		for _, s := range strings.Split(cells[1], "·") {
			if s = strings.Trim(strings.TrimSpace(s), "`"); s != "" {
				sites = append(sites, s)
			}
		}
		sort.Strings(sites)
		out[fp] = sites
	}
	return out
}

// covers reports whether a ratified row admits exactly this group. The
// fingerprint matching is not enough: the row names its sites and a group that
// has grown, shrunk or moved is a DIFFERENT exemption from the one approved.
func covers(sites []string, g group) (bool, string) {
	got := make([]string, 0, len(g.Sites))
	for _, s := range g.Sites {
		got = append(got, s.Func)
	}
	sort.Strings(got)
	if len(got) != len(sites) {
		return false, fmt.Sprintf("the exemption covers %d site(s), %d found", len(sites), len(got))
	}
	for i := range got {
		if got[i] != sites[i] {
			return false, fmt.Sprintf("the exemption names %v, found %v", sites, got)
		}
	}
	return true, ""
}

// runSelfTest proves the detector fires on a KNOWN duplicate and stays quiet on
// a known non-duplicate. Without both halves a detector that returns nothing
// looks exactly like a codebase with no duplication — which is the state metric
// D has been reported in twice, by instruments nobody kept.
func runSelfTest(min int) int {
	dir, err := os.MkdirTemp("", "dupes-selftest")
	if err != nil {
		fmt.Fprintln(os.Stderr, "self-test: cannot create fixture:", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// TWO COPIES OF ONE OPERATION, renamed — the shape that must be caught.
	// Long enough to clear any sane floor, and identical only in STRUCTURE:
	// every identifier and literal differs.
	twin := `package p
func alpha(xs []int) int {
	total := 0
	for _, v := range xs {
		if v > 3 {
			total += v * 2
		} else {
			total -= v
		}
	}
	return total
}
func beta(ys []int) int {
	sum := 1
	for _, w := range ys {
		if w > 9 {
			sum += w * 7
		} else {
			sum -= w
		}
	}
	return sum
}
`
	if err := os.WriteFile(filepath.Join(dir, "twin.go"), []byte(twin), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "self-test:", err)
		return 1
	}
	got, err := scan(dir, min, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "self-test:", err)
		return 1
	}
	if len(got) != 1 || len(got[0].Sites) != 2 {
		fmt.Fprintf(os.Stderr, "dupes SELF-TEST FAILED: a renamed copy was not detected (%d groups)\n", len(got))
		return 1
	}

	// AND A NEGATIVE CONTROL: two functions that merely share a shape must not
	// be called duplicates once the floor is applied. A detector that fires on
	// everything is a detector that gets ignored.
	solo := `package p
func onlyOne(a int) int { return a + 1 }
func alsoOne(b int) int { return b + 2 }
`
	dir2, _ := os.MkdirTemp("", "dupes-neg")
	defer func() { _ = os.RemoveAll(dir2) }()
	_ = os.WriteFile(filepath.Join(dir2, "solo.go"), []byte(solo), 0o600)
	neg, err := scan(dir2, min, false)
	if err != nil || len(neg) != 0 {
		fmt.Fprintf(os.Stderr, "dupes SELF-TEST FAILED: trivial functions were reported as duplicates (%d groups)\n", len(neg))
		return 1
	}
	fmt.Println("dupes self-test: both controls fired (OK)")
	return 0
}
