// Command plancode refuses implementation code in design and plan documents
// (observer-maps FR-8.1; D-13, D-14).
//
// A plan names signatures, shapes, test descriptions and file paths. A fenced
// Go block in one may declare types, constants, variables without function
// values, and functions WITHOUT bodies; anything else is implementation
// written into a plan, where it is neither compiled nor tested and quietly
// becomes the design.
//
// WHY THE PARSER AND NOT A GREP. "Has a body" is a question about the syntax
// tree: a signature and a one-line function differ by a brace a regular
// expression cannot tell from a struct literal.
//
// SCOPE: every Markdown file under 06_docs/02_features/*/03-architecture-design
// and */04-development. Features that shipped before the rule existed are listed
// in exempt with their reason; a feature begun after it is never exempt.
package main

import (
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

// exempt lists the features that shipped before D-13 made code in a plan a
// defect, each with its reason. A feature begun after it is never exempt, and
// the list only shrinks.
var exempt = map[string]string{
	"0.15.0-pre-broadcaster-ui-improvements": "0.15.0, shipped before D-13",
	"0.16.0-broadcaster-ui":                  "0.16.0, shipped before D-13",
	"global-ticker":                          "shipped before D-13",
	"map-ready-geometry":                     "0.17.0, shipped before D-13",
	"multi-voice-support":                    "shipped before D-13; its prior-art code is the record of what was built",
	"seismic-data":                           "shipped before D-13",
	"severe-alerts-modals":                   "0.13.0, shipped before D-13",
	"watchpost-cli":                          "the first release, shipped before D-13",
	"watchpost-performance-quality-pass":     "shipped before D-13",
}

// featureOf is the feature directory a document belongs to.
func featureOf(rel string) string {
	rest := strings.TrimPrefix(filepath.ToSlash(rel), "06_docs/02_features/")
	if i := strings.Index(rest, "/"); i > 0 {
		return rest[:i]
	}
	return ""
}

// maxBlocks bounds the fenced blocks read from one document (P10-02).
const maxBlocks = 500

// Finding is one fenced Go block that is not declarations alone.
type Finding struct {
	File string
	Line int
	Why  string
}

// judge answers why a fenced Go block is implementation, or "" when it is
// declarations alone.
func judge(src string) string {
	f, err := parser.ParseFile(token.NewFileSet(), "", "package plan\n"+src, parser.ParseComments)
	if err != nil {
		return "not Go declarations (a statement, or code outside a declaration): write it as signatures and types"
	}
	why := ""
	ast.Inspect(f, func(n ast.Node) bool {
		if why != "" {
			return false
		}
		switch d := n.(type) {
		case *ast.FuncDecl:
			if d.Body != nil {
				why = "func " + d.Name.Name + " has a body"
			}
		case *ast.FuncLit:
			why = "a function literal"
		}
		return true
	})
	return why
}

// blocks returns each fenced Go block in a document, with the line its first
// source line is on.
func blocks(text string) ([]string, []int) {
	var srcs []string
	var lines []int
	in := false
	var cur []string
	start := 0
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case !in && trimmed == "```go":
			in, cur, start = true, nil, i+2
		case in && strings.HasPrefix(trimmed, "```"):
			in = false
			if len(srcs) < maxBlocks {
				srcs = append(srcs, strings.Join(cur, "\n"))
				lines = append(lines, start)
			}
		case in:
			cur = append(cur, line)
		}
	}
	return srcs, lines
}

// inScope reports whether a path is a design or plan document the rule covers.
func inScope(rel string) bool {
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, "06_docs/02_features/") || !strings.HasSuffix(rel, ".md") {
		return false
	}
	return strings.Contains(rel, "/03-architecture-design/") || strings.Contains(rel, "/04-development/")
}

func scan(root string) ([]Finding, int, error) {
	var out []Finding
	docs := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil || !inScope(rel) {
			return rerr
		}
		if _, ok := exempt[featureOf(rel)]; ok {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		docs++
		srcs, lines := blocks(string(b))
		for i, src := range srcs {
			if why := judge(src); why != "" {
				out = append(out, Finding{File: filepath.ToSlash(rel), Line: lines[i], Why: why})
			}
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, docs, err
}

// selfTest proves the instrument can fail: every planted body is refused and
// every planted signature passes.
func selfTest() error {
	refused := []string{
		"func Render(s Size) Frame {\n\treturn Frame{}\n}",
		"x := 1",
		"var f = func() {}",
		"func (m *Map) Purge() error { return nil }",
	}
	passed := []string{
		"func Render(s Size) (Frame, error)",
		"type Frame struct{ Lines []string; Changed, FrameTicks uint64 }",
		"const MaxFrames = 72",
		"type Source interface{ Name() string }",
		"func (m *Map) SetPlayback(p Playback) error // off by default",
	}
	for _, src := range refused {
		if judge(src) == "" {
			return fmt.Errorf("a body was not refused: %q", src)
		}
	}
	for _, src := range passed {
		if why := judge(src); why != "" {
			return fmt.Errorf("a declaration was refused (%s): %q", why, src)
		}
	}
	srcs, lines := blocks("text\n```go\nx := 1\n```\nmore\n```go\ntype T int\n```\n")
	if len(srcs) != 2 || lines[0] != 3 || lines[1] != 7 {
		return fmt.Errorf("fences read wrongly: %d blocks at %v", len(srcs), lines)
	}
	if featureOf("06_docs/02_features/observer-maps/04-development/x.md") != "observer-maps" {
		return fmt.Errorf("the feature is read wrongly")
	}
	if !inScope("06_docs/02_features/x/04-development/implementation-plan.md") || inScope("06_docs/02_features/x/08-reports/plan-report.md") {
		return fmt.Errorf("the scope is wrong")
	}
	return nil
}

func main() {
	root := flag.String("root", ".", "the tree to scan")
	self := flag.Bool("self-test", false, "prove the instrument can fail, then exit")
	flag.Parse()
	if *self {
		if err := selfTest(); err != nil {
			fmt.Println("plancode SELF-TEST FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("plancode self-test: bodies refused, signatures passed (OK)")
		return
	}
	found, docs, err := scan(*root)
	if err != nil {
		fmt.Println("plancode:", err)
		os.Exit(1)
	}
	if docs == 0 {
		fmt.Println("plancode: no design or plan document was read — the scope is broken, so a pass would prove nothing")
		os.Exit(1)
	}
	if len(found) > 0 {
		fmt.Printf("plancode: implementation code in a plan (D-13) — signatures, shapes and tests only:\n")
		for _, f := range found {
			fmt.Printf("  %s:%d: %s\n", f.File, f.Line, f.Why)
		}
		os.Exit(1)
	}
	fmt.Printf("plancode: %d design and plan document(s), no implementation code (%d feature(s) exempt, shipped before D-13)\n", docs, len(exempt))
}
