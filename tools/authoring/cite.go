package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/branden-thompson/watchpost/tools/ids/idset"
)

// citedAt is the IDs each package's comments and code carried at the base
// the branch is judged against: package directory → ID → present.
type citedAt map[string]map[string]bool

// baseCitations reads every ID the tree's Go files carried at base, one git
// call for the whole tree. A package's set is judged, not a file's, so a
// comment moved between files of one package keeps its citations.
func baseCitations(root, base string) (citedAt, error) {
	pattern := "(" + strings.Join(idset.Families, "|") + ")-[0-9]+"
	out, err := exec.Command("git", "-C", root, "grep", "-I", "-o", "-E", pattern, base, "--", "*.go").Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return citedAt{}, nil // git grep found nothing
		}
		return nil, fmt.Errorf("reading the citations at %s: %w", base, err)
	}
	at := citedAt{}
	for _, line := range strings.Split(string(out), "\n") { // bounded by the output (P10-02)
		parts := strings.SplitN(line, ":", 3) // base:path:match
		if len(parts) != 3 {
			continue
		}
		dir := path.Dir(parts[1])
		if at[dir] == nil {
			at[dir] = map[string]bool{}
		}
		at[dir][idKey(parts[2])] = true
	}
	return at, nil
}

// idKey is an ID's family and number, its decimal or letter aside: FR-3.2
// and FR-3 cite one requirement.
func idKey(id string) string {
	m := idset.ID.FindStringSubmatch(id)
	if m == nil {
		return id
	}
	return m[1] + "-" + m[2]
}

// mergeBase is the commit the branch is judged against: where HEAD left main,
// as the remote knows it, else as the local branch does.
func mergeBase(root string) (string, error) {
	for _, main := range []string{"origin/main", "main"} { // bounded by the two names (P10-02)
		out, err := exec.Command("git", "-C", root, "merge-base", "HEAD", main).Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}
	return "", fmt.Errorf("no merge-base with main: the citation rule needs the branch's history (CI checks out with fetch-depth 0)")
}

// releaseNamed is a release number, which qualifies every ID after it to the
// end of its sentence: "0.19.5 Q7.4 (JM-3, DC-14)".
var releaseNamed = regexp.MustCompile(`\b\d+\.\d+\.\d+\b`)

// sentenceEnd is a full stop that ends a sentence: followed by a space or the
// end of the line, so a release number's own dots are not one.
var sentenceEnd = regexp.MustCompile(`\.(?:\s|$)`)

// checkCitations reports a bare ID in a comment that its package did not
// carry at the base (AP-CITE-01). An ID is qualified when a release number
// comes before it in its sentence, across the lines of one comment, or when
// it carries a release's own prefix (MVS-D-80): either way a reader knows
// which release's row it is.
func checkCitations(fset *token.FileSet, f *ast.File, rel string, base citedAt) []Finding {
	if base == nil {
		return nil
	}
	known := base[path.Dir(rel)]
	var out []Finding
	for _, g := range f.Comments { // bounded by the file's comments (P10-02)
		qualified := false
		for _, c := range g.List { // bounded by the group (P10-02)
			for n, line := range strings.Split(c.Text, "\n") { // bounded by the comment (P10-02)
				var bare []string
				bare, qualified = bareIDs(line, qualified)
				for _, id := range bare { // bounded by the line (P10-02)
					if known[idKey(id)] {
						continue
					}
					out = append(out, Finding{
						Rule: "AP-CITE-01", File: rel, Line: fset.Position(c.Pos()).Line + n,
						Text: id + " is a new citation with no release",
						Why:  "every release numbers its IDs from 1, so name the release a new citation means (0.19.5 D-11): write it as \"0.19.5 " + id + "\"; 06_docs/ids.md lists each release's row",
					})
				}
			}
		}
	}
	return out
}

// bareIDs are the IDs on a comment line that no release qualifies and that
// carry no prefix of their own. qualified says whether the sentence the line
// opens in has named a release; the result says whether the line ends in one
// that has.
func bareIDs(line string, qualified bool) ([]string, bool) {
	type mark struct {
		at   int
		kind byte // 'r' a release, 'e' a sentence's end, 'i' an ID
		end  int
	}
	var marks []mark
	for _, l := range releaseNamed.FindAllStringIndex(line, -1) { // bounded by the line (P10-02)
		marks = append(marks, mark{l[0], 'r', l[1]})
	}
	for _, l := range sentenceEnd.FindAllStringIndex(line, -1) { // bounded by the line (P10-02)
		marks = append(marks, mark{l[0], 'e', l[1]})
	}
	for _, l := range idset.ID.FindAllStringIndex(line, -1) { // bounded by the line (P10-02)
		marks = append(marks, mark{l[0], 'i', l[1]})
	}
	sort.Slice(marks, func(i, j int) bool { return marks[i].at < marks[j].at })
	var out []string
	for _, m := range marks { // bounded by the line (P10-02)
		switch {
		case m.kind == 'r':
			qualified = true
		case m.kind == 'e':
			qualified = false
		case !qualified && (m.at == 0 || line[m.at-1] != '-'): // MVS-D-80 carries its release's prefix
			out = append(out, line[m.at:m.end])
		}
	}
	return out, qualified
}
