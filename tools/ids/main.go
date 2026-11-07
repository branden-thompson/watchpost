// Command ids writes 06_docs/ids.md: for every ID the record's tables define,
// each feature's row and its subject, so a bare citation in a comment can be
// looked up in every release that used the number (0.19.5 D-11).
//
//	go run ./tools/ids            # print the index
//	go run ./tools/ids -write     # rewrite 06_docs/ids.md
//
// A ROW DEFINES AN ID when the ID is the whole of its first cell, and so does a
// heading that opens with it, the form 0.16.0 rules in ("# D-55 — THE
// INJECTOR SHIPS"). A table or a paragraph that mentions an ID refers to it, and is
// not read. The index is
// checked like the P10 ledger mirror: TestTheIndexIsCurrent regenerates it and
// compares, in the docs lane as well as in verify, because a docs change is
// what moves it.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/tools/ids/idset"
)

// row is one table row that defines an ID.
type row struct {
	family, num, suffix string
	feature, file       string // the feature folder (or "" for 06_docs itself), and the file within it
	subject             string
}

func main() {
	write := flag.Bool("write", false, "rewrite 06_docs/ids.md instead of printing it")
	flag.Parse()
	text, err := build(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ids:", err)
		os.Exit(2)
	}
	if !*write {
		fmt.Print(text)
		return
	}
	if err := os.WriteFile(filepath.Join("06_docs", "ids.md"), []byte(text), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ids:", err)
		os.Exit(2)
	}
}

// build reads every Markdown file under root's 06_docs, the mutant corpus and
// the index itself aside, and returns the index.
func build(root string) (string, error) {
	docs := filepath.Join(root, "06_docs")
	var rows []row
	err := filepath.WalkDir(docs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "mutants" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") || filepath.Base(path) == "ids.md" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(docs, path)
		rows = append(rows, rowsOf(string(b), filepath.ToSlash(rel))...)
		return nil
	})
	if err != nil {
		return "", err
	}
	return render(rows, releases(docs)), nil
}

// definingCell is a first cell that is an ID and nothing else, emphasis aside.
var definingCell = regexp.MustCompile(`^(?:` + strings.Join(idset.Families, "|") + `)-\d+(?:\.\d+)*[a-z]?$`)

// definingHeading is a heading that opens with an ID, and its title.
var definingHeading = regexp.MustCompile(`^#{1,4}\s+(?:\*\*)?((?:` + strings.Join(idset.Families, "|") + `)-\d+(?:\.\d+)*[a-z]?)(?:\*\*)?(?:\s*[—:–-]\s*|\s+|$)(.*)$`)

// rowsOf is the rows and headings in one file that define an ID, the first
// per ID: a second one for the same ID in one file refers back.
func rowsOf(src, rel string) []row {
	feature, file := "", rel
	if f, rest, ok := strings.Cut(rel, "/"); ok && f == "02_features" {
		feature, file, _ = strings.Cut(rest, "/")
	}
	seen := map[string]bool{}
	var out []row
	for _, line := range strings.Split(src, "\n") { // bounded by the file (P10-02)
		cells := cellsOf(line)
		if h := definingHeading.FindStringSubmatch(line); h != nil {
			cells = []string{h[1], strings.NewReplacer("**", "", "`", "").Replace(h[2])}
		}
		if len(cells) < 2 || !definingCell.MatchString(cells[0]) || seen[cells[0]] {
			continue
		}
		seen[cells[0]] = true
		m := idset.ID.FindStringSubmatch(cells[0])
		out = append(out, row{family: m[1], num: m[2], suffix: m[3], feature: feature, file: file, subject: subjectOf(cells[1:])})
	}
	return out
}

// cellsOf splits a Markdown table row into its cells, each stripped of
// emphasis and code marks; a line that is not a row has none.
func cellsOf(line string) []string {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "|") || !strings.HasSuffix(t, "|") {
		return nil
	}
	parts := strings.Split(strings.Trim(t, "|"), "|")
	for i, p := range parts { // bounded by the row (P10-02)
		parts[i] = strings.TrimSpace(strings.NewReplacer("**", "", "`", "", "*", "").Replace(p))
	}
	return parts
}

// date is a cell that is a date and nothing else.
var date = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// subjectOf is a row's subject: its first cell after the ID that says
// something, not a date, cut to a line.
func subjectOf(cells []string) string {
	for _, c := range cells { // bounded by the row (P10-02)
		if c == "" || date.MatchString(c) {
			continue
		}
		r := []rune(strings.Join(strings.Fields(c), " "))
		if len(r) > 110 {
			return string(r[:109]) + "…"
		}
		return string(r)
	}
	return ""
}

// versionIn finds a release number such as 0.19.5.
var versionIn = regexp.MustCompile(`\b\d+\.\d+\.\d+\b`)

// releases is each feature folder's release, where the record states one: in
// the folder's name, else in its project brief's title.
func releases(docs string) map[string]string {
	out := map[string]string{}
	entries, _ := os.ReadDir(filepath.Join(docs, "02_features"))
	for _, e := range entries { // bounded by the features (P10-02)
		if v := versionIn.FindString(e.Name()); v != "" && strings.HasPrefix(e.Name(), v) {
			out[e.Name()] = v
			continue
		}
		b, err := os.ReadFile(filepath.Join(docs, "02_features", e.Name(), "01-objectives", "project-brief.md"))
		if err != nil {
			continue
		}
		for _, line := range strings.SplitN(string(b), "\n", 12) { // the front matter (P10-02)
			if strings.HasPrefix(line, "title:") {
				out[e.Name()] = versionIn.FindString(line)
			}
		}
	}
	return out
}

// render writes the index: a section per family, in idset's order, and in
// each the IDs by number, every feature's row under its ID.
func render(rows []row, rel map[string]string) string {
	order := map[string]int{}
	for i, f := range idset.Families { // bounded by the families (P10-02)
		order[f] = i
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.family != b.family {
			return order[a.family] < order[b.family]
		}
		if na, nb := atoi(a.num), atoi(b.num); na != nb {
			return na < nb
		}
		if a.suffix != b.suffix {
			return a.suffix < b.suffix
		}
		if a.feature != b.feature {
			return a.feature < b.feature
		}
		return a.file < b.file
	})
	var sb strings.Builder
	sb.WriteString(header)
	family := ""
	for _, r := range rows { // bounded by the rows (P10-02)
		if r.family != family {
			family = r.family
			fmt.Fprintf(&sb, "\n## %s\n\n| ID | Release | Where | Subject |\n|---|---|---|---|\n", family)
		}
		where := "`" + r.file + "`"
		if r.feature != "" {
			where = "`" + r.feature + "/" + r.file + "`"
		}
		fmt.Fprintf(&sb, "| %s-%s%s | %s | %s | %s |\n", r.family, r.num, r.suffix, rel[r.feature], where, strings.ReplaceAll(r.subject, "|", "\\|"))
	}
	return sb.String()
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s) // the pattern matched digits
	return n
}

const header = `# ID index

> **Generated by ` + "`go run ./tools/ids -write`" + `; do not edit it by hand.** ` + "`TestTheIndexIsCurrent`" + `
> fails when it is stale, in the docs lane and in verify.

Each release numbers its rulings, requirements and findings from 1, so a bare ID such as ` + "`D-124`" + `
can mean a different row in each release that used it (0.19.5 D-11). This index lists, for every ID
the record's tables define, each release's row and its subject. Where says the feature folder under
` + "`06_docs/02_features/`" + ` and the file; Release is the folder's version where the record states one.

**To read an old citation**, find the ID here and take the release of the code's own history: the
file's ` + "`git log`" + ` says which release wrote the comment. **A new citation names its release**
(` + "`0.19.5 D-4`" + `, or a release's own prefix such as ` + "`MVS-D-80`" + `), which ` + "`tools/authoring`" + ` holds
over the lines a branch adds (` + "`AP-CITE-01`" + `). A row defines an ID when the ID is the whole of its first
cell; a later table that mentions it is not listed.
`
