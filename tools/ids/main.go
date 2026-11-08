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
// not read. The multi-voice release's rulings were never tabled, so an MVS-D
// ID is indexed by its first mention instead (0.19.5 D-35). The index is
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
	"unicode"

	"github.com/branden-thompson/watchpost/tools/ids/idset"
)

// row is one table row that defines an ID.
type row struct {
	family, num, suffix string
	feature, file       string // the feature folder (or "" for 06_docs itself), and the file within it
	line                int    // the mention's line, for an MVS-D row; 0 for a defining row
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
	mvs := map[string]mention{}
	untabled := map[string]string{} // the untabledFeature folder's files, by path, read after every row is known
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
		mentionsOf(string(b), filepath.ToSlash(rel), mvs)
		if strings.HasPrefix(filepath.ToSlash(rel), "02_features/"+untabledFeature+"/") {
			untabled[filepath.ToSlash(rel)] = string(b)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	rows = append(rows, untabledRows(rows, untabled)...)
	return render(rows, releases(docs)) + renderMVS(mvs, releases(docs)), nil
}

// untabledFeature is the release whose later rulings were never tabled:
// its D-IDs past its tables are cited in code and mentioned in its docs
// (0.19.5 D-37).
const untabledFeature = "0.16.0-broadcaster-ui"

// dCite is a D-ID where it stands in a line; dBold is one in bold.
var (
	dCite = regexp.MustCompile(`\bD-(\d+)\b`)
	dBold = regexp.MustCompile(`\*\*D-(\d+)\*\*`)
)

// untabledRows is a row for each D-ID the untabled release's docs mention
// and its tables do not define, at its best mention there: the first line
// that states it in bold, else the first that names it. A mention another
// release qualifies ("MVS-D-80", "0.18.0 D-97", "watchpost-cli D-124") is that
// release's, not this one's. The subject says it is a mention.
func untabledRows(rows []row, files map[string]string) []row {
	defined := map[string]bool{}
	for _, r := range rows { // bounded by the rows (P10-02)
		if r.feature == untabledFeature && r.family == "D" {
			defined[r.num] = true
		}
	}
	paths := make([]string, 0, len(files))
	for p := range files { // bounded by the folder (P10-02)
		paths = append(paths, p)
	}
	sort.Strings(paths)
	best := map[string]mention{}
	for _, p := range paths { // bounded by the folder (P10-02)
		_, rest, _ := strings.Cut(p, "/")
		_, file, _ := strings.Cut(rest, "/")
		for i, line := range strings.Split(files[p], "\n") { // bounded by the file (P10-02)
			bold := map[string]bool{}
			for _, b := range dBold.FindAllStringSubmatch(line, -1) { // bounded by the line (P10-02)
				bold[b[1]] = true
			}
			for _, m := range dCite.FindAllStringSubmatchIndex(line, -1) { // bounded by the line (P10-02)
				num := line[m[2]:m[3]]
				if defined[num] || qualified(line[:m[0]]) {
					continue
				}
				rank := rankCited
				if bold[num] {
					rank = rankRuled
				}
				keep(best, num, mention{rank: rank, row: row{family: "D", num: num, feature: untabledFeature, file: file, line: i + 1, subject: subjectOf([]string{"Mentioned: " + plain(line)})}})
			}
		}
	}
	out := make([]row, 0, len(best))
	for _, m := range best { // bounded by the IDs (P10-02)
		out = append(out, m.row)
	}
	return out
}

// qualified reports whether the text before an ID names another release:
// a prefix, a version or the CLI's rulings.
func qualified(before string) bool {
	t := strings.TrimRight(before, " *`(")
	return strings.HasSuffix(before, "MVS-") || strings.HasSuffix(t, "watchpost-cli") || strings.HasSuffix(t, "cli") || versionIn.MatchString(lastWord(t))
}

// lastWord is t's last space-separated word.
func lastWord(t string) string {
	if i := strings.LastIndexByte(t, ' '); i >= 0 {
		return t[i+1:]
	}
	return t
}

// mvsFeature is the multi-voice release's folder, where its rulings are
// first mentioned.
const mvsFeature = "multi-voice-support"

// mvsCite is an MVS-D ID wherever it stands in a line.
var mvsCite = regexp.MustCompile(`\bMVS-D-(\d+)\b`)

// mvsHeading is a heading that opens with an MVS-D ID, and its title: the
// form MVS-D-88 to 92 have, one file each.
var mvsHeading = regexp.MustCompile(`^#{1,4}\s+MVS-D-(\d+)\s*[—:–-]\s*(.*)$`)

// mvsBold is an MVS-D ID in bold and what it ruled, the form of the project
// brief's Rulings row ("**MVS-D-4** full maritime report · …") and of a
// build log's "**MVS-D-80 — the inter-card transition rule**": the title
// inside the bold, else the words after it up to the next "·" or cell.
var mvsBold = regexp.MustCompile(`\*\*MVS-D-(\d+)(?:\s*[—:–-]\s*([^*]+))?\*\*([^·|]*)`)

// aside is a lead parenthesis such as "(E-1)" or "(2026-08-29)", which names
// a source or a date, not what was ruled.
var aside = regexp.MustCompile(`^\s*\([^)]*\)`)

// The ranks of an MVS-D mention, best first.
const (
	rankHeading   = iota // its own file's heading
	rankRuledHere        // a bold statement of what it ruled, in the multi-voice docs
	rankRuled            // a bold statement elsewhere in the record
	rankCitedHere        // any line in the multi-voice docs
	rankCited            // any line elsewhere
)

// mention is one MVS-D ID's best mention so far, and its rank.
type mention struct {
	rank int
	row  row
}

// mentionsOf records, for each MVS-D ID in one file, its mention when it
// outranks the one held. The walk is in path order and a file is read top
// down, so among equal ranks the first one met stays.
func mentionsOf(src, rel string, into map[string]mention) {
	feature, file := "", rel
	if f, rest, ok := strings.Cut(rel, "/"); ok && f == "02_features" {
		feature, file, _ = strings.Cut(rest, "/")
	}
	ruled, cited := rankRuled, rankCited
	if feature == mvsFeature {
		ruled, cited = rankRuledHere, rankCitedHere
	}
	at := func(num string, rank, line int, subject string) {
		keep(into, num, mention{rank: rank, row: row{family: "MVS-D", num: num, feature: feature, file: file, line: line, subject: subjectOf([]string{subject})}})
	}
	for i, line := range strings.Split(src, "\n") { // bounded by the file (P10-02)
		if h := mvsHeading.FindStringSubmatch(line); h != nil {
			at(h[1], rankHeading, i+1, plain(h[2]))
			continue
		}
		for _, b := range mvsBold.FindAllStringSubmatch(line, -1) { // bounded by the line (P10-02)
			if what := statement(b, line); what != "" {
				at(b[1], ruled, i+1, what)
			}
		}
		for _, m := range mvsCite.FindAllStringSubmatch(line, -1) { // bounded by the line (P10-02)
			at(m[1], cited, i+1, plain(line))
		}
	}
}

// statement is what a bold MVS-D says it ruled: the title inside the bold,
// else the words after it, a lead "(E-1)" aside; in a table row with nothing
// after the ID, the row's later cells. It is "" when none of those says
// anything, which leaves the line a mention. Words that open with ")", ","
// or ";" continue a sentence the ID was cited in, so they are not a statement.
func statement(b []string, line string) string {
	if t := plain(b[2]); t != "" {
		return t
	}
	after := strings.TrimSpace(b[3])
	if strings.HasPrefix(after, ")") || strings.HasPrefix(after, ",") || strings.HasPrefix(after, ";") {
		return ""
	}
	if t := plain(aside.ReplaceAllString(strings.TrimLeft(after, ":—–- "), "")); says(t) {
		return t
	}
	cells := cellsOf(line)
	for k, c := range cells { // bounded by the row (P10-02)
		if strings.Contains(c, "MVS-D-"+b[1]) && k+1 < len(cells) {
			return strings.Join(cells[k+1:], " — ")
		}
	}
	return ""
}

// says reports whether t has three words or more with a letter in them.
func says(t string) bool {
	n := 0
	for _, w := range strings.Fields(t) { // bounded by the text (P10-02)
		if strings.IndexFunc(w, unicode.IsLetter) >= 0 {
			n++
		}
	}
	return n >= 3
}

// keep holds m for its ID unless a mention of the same or a better rank is
// already held.
func keep(into map[string]mention, num string, m mention) {
	if held, ok := into[num]; ok && held.rank <= m.rank {
		return
	}
	into[num] = m
}

// plain is a Markdown line as words: its heading marks, table bars, emphasis
// and code marks removed.
func plain(line string) string {
	s := strings.NewReplacer("**", "", "`", "", "*", "", "|", " ").Replace(line)
	return strings.TrimLeft(strings.TrimSpace(s), "#> -")
}

// renderMVS writes the MVS-D section: each ID by number, where it is first
// mentioned, and the sentence there.
func renderMVS(mvs map[string]mention, rel map[string]string) string {
	if len(mvs) == 0 {
		return ""
	}
	nums := make([]string, 0, len(mvs))
	for n := range mvs { // bounded by the IDs (P10-02)
		nums = append(nums, n)
	}
	sort.Slice(nums, func(i, j int) bool { return atoi(nums[i]) < atoi(nums[j]) })
	var sb strings.Builder
	sb.WriteString(mvsHeader)
	for _, n := range nums { // bounded by the IDs (P10-02)
		r := mvs[n].row
		where := r.file
		if r.feature != "" {
			where = r.feature + "/" + r.file
		}
		fmt.Fprintf(&sb, "| MVS-D-%s | %s | `%s:%d` | %s |\n", r.num, rel[r.feature], where, r.line, strings.ReplaceAll(r.subject, "|", "\\|"))
	}
	return sb.String()
}

const mvsHeader = `
## MVS-D

The multi-voice release ruled in conversation and cited its rulings as MVS-D-n, but never put them
in a table, so each one here points at its first mention instead (0.19.5 D-35): its own file's
heading where it has one, else its earliest line in ` + "`multi-voice-support/`" + `, else its earliest
line elsewhere in the record. The subject is that line, which is where its context is.

| ID | Release | Where | Subject |
|---|---|---|---|
`

// definingCell is a first cell that is an ID and nothing else, emphasis
// aside, or an ID and one parenthesis, as a requirement carries its priority
// (0.18.0 FR-3.8 is written "(P1)" after its ID).
var definingCell = regexp.MustCompile(`^(?:` + strings.Join(idset.Families, "|") + `)-\d+(?:\.\d+)*[a-z]?(?:\s*\([^)]*\))?$`)

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

// subjectOf is a row's subject: its first cell after the ID with a word in
// it, cut to a line. A cell of references only ("W1.13, W3.7") or a date
// says nothing; when no cell has a word, the first that is not empty or a
// date stands.
func subjectOf(cells []string) string {
	pick := ""
	for _, c := range cells { // bounded by the row (P10-02)
		if c == "" || date.MatchString(c) {
			continue
		}
		if pick == "" {
			pick = c
		}
		if word.MatchString(c) {
			pick = c
			break
		}
	}
	r := []rune(strings.Join(strings.Fields(pick), " "))
	if len(r) > 110 {
		return string(r[:109]) + "…"
	}
	return string(r)
}

// word is three lower-case letters together, which a reference such as
// "W1.13" or "0.19.5 D-4" never has.
var word = regexp.MustCompile(`\p{Ll}{3}`)

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
		where := r.file
		if r.feature != "" {
			where = r.feature + "/" + r.file
		}
		if r.line > 0 {
			where += ":" + strconv.Itoa(r.line)
		}
		where = "`" + where + "`"
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
