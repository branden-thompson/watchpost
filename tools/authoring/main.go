// Command authoring checks the A2DH code-authoring rules that a machine can
// decide, over the whole tree.
//
// WHY IT IS A TOOL AND NOT A GREP. Two of the three rules here are questions
// about the SYNTAX TREE: whether a match is in a comment or a string literal,
// and which declaration a comment block binds to. The second cannot be asked of
// text at all — a doc comment that fuses with its neighbour's reads as two
// healthy comments to every line-based checker, while the function above it has
// silently lost its contract.
//
// AND BECAUSE AN INSTRUMENT MUST BE ABLE TO PROVE IT CAN FAIL. `-self-test`
// runs planted specimens — including the catalogue's own anti-specimens — and
// reports a failure if any goes undetected, or if a corrected form is flagged.
// The shell version this replaces was validated once, by hand, and that proof
// did not survive the session.
//
// SCOPED TO THE ARTEFACT, NEVER TO THE SESSION (HUM LEAD, 2026-09-16). An
// anti-pattern is not excused by predating the change that finds it. A check
// scoped to authorship can be silenced by narrowing it, which is a failure mode
// this repository has already seen.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// skipDir names the trees this tool must not judge.
//
// `third_party` IS VENDORED AND NOT OURS TO REWRITE; its gaps go upstream.
// `06_docs/mutants` holds deliberate defects, and a mutant's whole purpose is to
// describe what the code would do if it were wrong.
func skipDir(base string) bool {
	switch base {
	case ".git", "third_party", "mutants", "dist", "node_modules":
		return true
	// THIS TOOL'S OWN SOURCE holds the anti-specimens as data — the phrase table
	// and the self-test's planted comments. A detector that flagged its own
	// detector would be reporting its correctness as a defect, and the specimens
	// are exactly what must NOT be edited away.
	case "authoring":
		return true
	}
	return false
}

func scan(root string) ([]Finding, int, error) {
	var out []Finding
	files := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDir(filepath.Base(path)) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // unparseable: not this tool's complaint
		}
		files++
		rel := strings.TrimPrefix(strings.TrimPrefix(path, root), "/")
		out = append(out, checkHistory(fset, f, rel)...)
		out = append(out, checkBlankKeepAlive(fset, f, rel)...)
		out = append(out, checkDocAttached(fset, f, rel)...)
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, files, err
}

// report prints the findings grouped by rule, longest-standing rule first.
//
// EXTRACTED AT THE STATEMENT CEILING (P10-04). `main` is argument parsing, one
// branch per output mode, and an exit code; the grouping and the printing are a
// second job that was making it a page and a half.
//
// THE `Why` COMES FROM THE FIRST FINDING because it is a property of the RULE,
// not of the site — every finding a rule emits carries the same remedy, and
// printing it once per rule is what keeps a 125-finding run readable.
func report(found []Finding, limit int) {
	byRule := map[string][]Finding{}
	for _, f := range found { // bounded by the findings (P10-02)
		byRule[f.Rule] = append(byRule[f.Rule], f)
	}
	rules := make([]string, 0, len(byRule))
	for r := range byRule { // bounded by the catalogue (P10-02)
		rules = append(rules, r)
	}
	sort.Strings(rules)

	for _, r := range rules { // bounded by the catalogue (P10-02)
		fs := byRule[r]
		fmt.Printf("authoring: %s — %d finding(s)\n", r, len(fs))
		for i, f := range fs { // bounded by the findings (P10-02)
			if i >= limit {
				fmt.Printf("    … and %d more\n", len(fs)-limit)
				break
			}
			fmt.Printf("    %s:%d  %s\n", f.File, f.Line, f.Text)
		}
		fmt.Printf("    -> %s\n", fs[0].Why)
	}
}

func main() {
	root := flag.String("root", ".", "the tree to scan")
	asJSON := flag.Bool("json", false, "machine-readable output")
	selfTest := flag.Bool("self-test", false, "prove the instrument can fail, then exit")
	limit := flag.Int("limit", 15, "findings printed per rule before summarising")
	flag.Parse()

	if *selfTest {
		os.Exit(runSelfTest())
	}

	found, files, err := scan(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "authoring: %v\n", err)
		os.Exit(2)
	}
	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(found)
		if len(found) > 0 {
			os.Exit(1)
		}
		return
	}

	report(found, *limit)

	if len(found) > 0 {
		fmt.Fprintf(os.Stderr, "\nauthoring: %d finding(s) across %d Go file(s).\n", len(found), files)
		fmt.Fprintln(os.Stderr, "  Each rule, with examples of the accepted form, is in 06_docs/code-standards.md.")
		os.Exit(1)
	}
	fmt.Printf("authoring: OK — %d Go file(s), no findings\n", files)
	fmt.Println("  scope: the WHOLE tree, less vendored third_party and the mutant corpus.")
	fmt.Println("  It decides three rules a syntax tree can decide. It cannot see a comment that is")
	fmt.Println("  merely WRONG — the larger class, which still needs a reader.")
	fmt.Println("  The rules are written out in 06_docs/code-standards.md.")
}
