package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// changedLines is each Go file's lines the branch changed since base, from
// the working tree: path → line → changed.
type changedLines map[string]map[int]bool

// branchChanges reads the changed lines of every Go file since base, one git
// call for the tree. A new file needs `git add -N` to be seen, as the gate's
// other tracked-file checks do.
func branchChanges(root, base string) (changedLines, error) {
	out, err := exec.Command("git", "-C", root, "diff", "-U0", "--no-color", base, "--", "*.go").Output()
	if err != nil {
		return nil, fmt.Errorf("reading the changes since %s: %w", base, err)
	}
	ch := changedLines{}
	file := ""
	for _, line := range strings.Split(string(out), "\n") { // bounded by the diff (P10-02)
		if name, ok := strings.CutPrefix(line, "+++ b/"); ok {
			file = name
			continue
		}
		if m := hunk.FindStringSubmatch(line); m != nil && file != "" {
			start, _ := strconv.Atoi(m[1]) // the pattern matched digits
			n := 1
			if m[2] != "" {
				n, _ = strconv.Atoi(m[2])
			}
			if ch[file] == nil {
				ch[file] = map[int]bool{}
			}
			for l := start; l < start+n; l++ { // bounded by the hunk (P10-02)
				ch[file][l] = true
			}
		}
	}
	return ch, nil
}

// hunk is a unified diff hunk's header, and the new side's start and length.
var hunk = regexp.MustCompile(`^@@ -\S+ \+(\d+)(?:,(\d+))? @@`)

// capsContinuation is a paragraph that opens with an all-caps AND, BUT or SO
// run: "AND THE REFUSAL IS CARRIED".
var capsContinuation = regexp.MustCompile(`^//\s+(?:AND|BUT|SO)(?: [A-Z][A-Z'’-]*)+\b`)

// checkCapsContinuation reports, as a soft warning, a comment paragraph that
// opens with an all-caps AND, BUT or SO inside a declaration the branch
// changed (AP-CAPS-01, 0.19.5 D-12). Such a paragraph means something only
// after the one above it, and loses its sense when either moves. Only changed
// declarations are judged: the rule is brought to code as it is edited.
func checkCapsContinuation(fset *token.FileSet, f *ast.File, rel string, changed changedLines) []Finding {
	lines := changed[rel]
	if lines == nil {
		return nil
	}
	var out []Finding
	for _, d := range f.Decls { // bounded by the file's declarations (P10-02)
		from, to := declSpan(fset, d)
		if !touches(lines, from, to) {
			continue
		}
		for _, g := range f.Comments { // bounded by the file's comments (P10-02)
			if at := fset.Position(g.Pos()).Line; at < from || at > to {
				continue
			}
			for n, c := range g.List { // bounded by the group (P10-02)
				opens := n == 0 || strings.TrimSpace(g.List[n-1].Text) == "//"
				if opens && capsContinuation.MatchString(c.Text) {
					out = append(out, Finding{
						Rule: "AP-CAPS-01", File: rel, Line: fset.Position(c.Pos()).Line,
						Text: strings.TrimSpace(c.Text),
						Why:  "a paragraph that opens AND, BUT or SO means something only after the one above it: say what it adds in sentence case, standing alone",
					})
				}
			}
		}
	}
	return out
}

// declSpan is a declaration's lines, its doc included.
func declSpan(fset *token.FileSet, d ast.Decl) (from, to int) {
	from = fset.Position(d.Pos()).Line
	switch x := d.(type) {
	case *ast.FuncDecl:
		if x.Doc != nil {
			from = fset.Position(x.Doc.Pos()).Line
		}
	case *ast.GenDecl:
		if x.Doc != nil {
			from = fset.Position(x.Doc.Pos()).Line
		}
	}
	return from, fset.Position(d.End()).Line
}

// touches reports whether any line from..to changed.
func touches(lines map[int]bool, from, to int) bool {
	for l := from; l <= to; l++ { // bounded by the declaration (P10-02)
		if lines[l] {
			return true
		}
	}
	return false
}
