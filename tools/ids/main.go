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
	all := map[string]string{}      // every file, by path, for the follow-ups only mentioned
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
		if !aboutIDs(filepath.ToSlash(rel)) {
			rows = append(rows, rowsOf(string(b), filepath.ToSlash(rel))...)
		}
		mentionsOf(string(b), filepath.ToSlash(rel), mvs)
		if strings.HasPrefix(filepath.ToSlash(rel), "02_features/"+untabledFeature+"/") {
			untabled[filepath.ToSlash(rel)] = string(b)
		}
		all[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		return "", err
	}
	rows = append(rows, untabledRows(rows, untabled)...)
	rows = append(rows, followUpRows(rows, all)...)
	rows = append(rows, findingRows(all)...)
	rows = append(rows, samRows(rows, all)...)
	rows = append(rows, taskRows(rows, all)...)
	return render(rows, releases(docs)) + renderMVS(mvs, releases(docs)), nil
}

// aboutIDs reports whether a file is a blind trace's or its key's, 0.19.5's
// M5 or 0.19.6's M4: those list the IDs a reviewer met and whether each
// resolved, in tables led by the ID, so they define none and their mentions
// are no follow-up's best.
func aboutIDs(rel string) bool {
	return strings.Contains(rel, "/08-reports/m5-") || strings.Contains(rel, "/08-reports/m4-")
}

// readable is files less the blind traces and their keys, the files a
// mention is read from.
func readable(files map[string]string) map[string]string {
	read := map[string]string{}
	for p, src := range files { // bounded by the record (P10-02)
		if !aboutIDs(p) {
			read[p] = src
		}
	}
	return read
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
	return mentionRows(files, "D", "", dCite, dBold, defined)
}

// fCite is an F-ID where it stands in a line; fBold is one in bold.
var (
	fCite = regexp.MustCompile(`\bF-(\d+)\b`)
	fBold = regexp.MustCompile(`\*\*F-(\d+)\*\*`)
)

// followUpRows is a row for each follow-up the record mentions and no table
// defines, at its best mention (0.19.5 D-38). Follow-ups are numbered once
// for the whole project, so an F-ID any row defines, a closed one in
// follow-ups-closed.md included, is defined. The blind traces and their keys
// are left out (aboutIDs).
func followUpRows(rows []row, files map[string]string) []row {
	defined := map[string]bool{}
	for _, r := range rows { // bounded by the rows (P10-02)
		if r.family == "F" {
			defined[r.num] = true
		}
	}
	return mentionRows(readable(files), "F", "", fCite, fBold, defined)
}

// mentionRows is the best mention of each ID of a family that defined does
// not hold, over files by path: a line that states it in bold, else the
// first line that names it, in path order. A mention another release
// qualifies is that release's; one qualified by own, the family's own
// release, is the family's.
func mentionRows(files map[string]string, family, own string, cite, bold *regexp.Regexp, defined map[string]bool) []row {
	paths := make([]string, 0, len(files))
	for p := range files { // bounded by the files (P10-02)
		paths = append(paths, p)
	}
	sort.Strings(paths)
	best := map[string]mention{}
	for _, p := range paths { // bounded by the files (P10-02)
		feature, file := "", p
		if f, rest, ok := strings.Cut(p, "/"); ok && f == "02_features" {
			feature, file, _ = strings.Cut(rest, "/")
		}
		for i, line := range strings.Split(files[p], "\n") { // bounded by the file (P10-02)
			bolded := map[string]bool{}
			for _, b := range bold.FindAllStringSubmatch(line, -1) { // bounded by the line (P10-02)
				bolded[b[1]] = true
			}
			for _, m := range cite.FindAllStringSubmatchIndex(line, -1) { // bounded by the line (P10-02)
				num := line[m[2]:m[3]]
				if defined[num] || qualified(line[:m[0]]) && (own == "" || !strings.HasSuffix(strings.TrimRight(line[:m[0]], " *`("), own)) {
					continue
				}
				rank := rankCited
				if bolded[num] {
					rank = rankRuled
				}
				keep(best, num, mention{rank: rank, row: row{family: family, num: num, feature: feature, file: file, line: i + 1, subject: subjectOf([]string{"Mentioned: " + plain(line)})}})
			}
		}
	}
	out := make([]row, 0, len(best))
	for _, m := range best { // bounded by the IDs (P10-02)
		out = append(out, m.row)
	}
	return out
}

// sfFamily is 0.9.0's red team's findings ("red-team 0.9.0 S-F10"), a release
// older than the record, so no table defines one: each is indexed at its best
// mention, the blind traces and their keys aside (0.19.6 D-50). It is written
// with no hyphen before its number.
const sfFamily = "S-F"

// sfRelease is the release whose red team numbered the S-F findings.
const sfRelease = "0.9.0"

// sfCite is an S-F ID where it stands in a line; sfBold is one in bold.
var (
	sfCite = regexp.MustCompile(`\bS-F(\d+)\b`)
	sfBold = regexp.MustCompile(`\*\*S-F(\d+)\*\*`)
)

// findingRows is a row for each S-F ID the record mentions, at its best mention.
func findingRows(files map[string]string) []row {
	return mentionRows(readable(files), sfFamily, sfRelease, sfCite, sfBold, map[string]bool{})
}

// samRows is a row for each SAM-D ruling the record mentions and its tables
// do not define, at its best mention.
func samRows(rows []row, files map[string]string) []row {
	defined := map[string]bool{}
	for _, r := range rows { // bounded by the rows (P10-02)
		if r.family == samFamily {
			defined[r.num] = true
		}
	}
	return mentionRows(readable(files), samFamily, "", samCite, samBold, defined)
}

// taskRows is a row for each plan task the record mentions and no table
// defines, at its best mention: a task a build log's batch names and the plan
// never tabled (0.18.0's W18.4).
func taskRows(rows []row, files map[string]string) []row {
	defined := map[string]bool{}
	for _, r := range rows { // bounded by the rows (P10-02)
		if r.family == idset.Task {
			defined[r.num+r.suffix] = true
		}
	}
	out := mentionRows(readable(files), idset.Task, "", taskCite, taskBold, defined)
	for i := range out { // bounded by the mentions (P10-02)
		n, s, _ := strings.Cut(out[i].num, ".")
		out[i].num, out[i].suffix = n, "."+s // the wave, and the task in it
	}
	return out
}

// taskCite is a plan task where it stands in a line, wave and number the
// first group; taskBold is one in bold.
var (
	taskCite = regexp.MustCompile(`\b` + idset.Task + `(\d+(?:\.\d+)+)\b`)
	taskBold = regexp.MustCompile(`\*\*` + idset.Task + `(\d+(?:\.\d+)+)\*\*`)
)

// samCite is a SAM-D ruling where it stands in a line; samBold is one in bold.
var (
	samCite = regexp.MustCompile(`\b` + samFamily + `-(\d+)\b`)
	samBold = regexp.MustCompile(`\*\*` + samFamily + `-(\d+)\*\*`)
)

// written is a row's ID as the record writes it.
func written(r row) string {
	if r.family == sfFamily {
		return r.family + r.num + r.suffix
	}
	return idset.Written(r.family, r.num, r.suffix)
}

// familyPatterns are idset's families, each matching a family whole.
var familyPatterns = func() []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(idset.Families))
	for i, p := range idset.Families { // bounded by the families (P10-02)
		out[i] = regexp.MustCompile(`^(?:` + p + `)$`)
	}
	return out
}()

// familyOrder is the index's order of sections: idset's families, a UAT
// round's under its pattern, then the task, SAM-D and S-F.
func familyOrder(f string) int {
	for i, p := range familyPatterns { // bounded by the families (P10-02)
		if p.MatchString(f) {
			return i
		}
	}
	return len(idset.Families) + strings.Index(" "+idset.Task+" "+samFamily+" "+sfFamily+" ", " "+f+" ")
}

// qualified reports whether the text before an ID names another release:
// a prefix, a version or the CLI's rulings.
func qualified(before string) bool {
	t := strings.TrimRight(before, " *`(")
	return strings.HasSuffix(before, "MVS-") || strings.HasSuffix(t, "watchpost-cli") || strings.HasSuffix(t, "cli") || idset.Release.MatchString(lastWord(t))
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
// aside, or an ID and a code in parentheses: a requirement's priority or tier
// (0.18.0 FR-3.8 is written "(P1)" after its ID) or a release. Words in the
// parentheses ("cited") make the cell a note about the ID, not its row.
var definingCell = regexp.MustCompile(`^` + idText + `(?:\s*\((?:[A-Z]\d+|\d+\.\d+\.\d+)\))?$`)

// idText is an ID's text in either of idset's forms, or a ruling of the
// severe-alerts release, which carries its release in its prefix as MVS-D
// does and, unlike MVS-D, tabled its rulings (0.19.6 D-50).
var idText = `(?:(?:` + strings.Join(idset.Families, "|") + `)-\d+(?:\.\d+)*[a-z]?|` + idset.Task + `\d+(?:\.\d+)+|` + samFamily + `-\d+)`

// samFamily is the severe-alerts release's prefix for its rulings.
const samFamily = "SAM-D"

// samID is a SAM-D ruling, its number the first group.
var samID = regexp.MustCompile(`^` + samFamily + `-(\d+)$`)

// definingHeading is a heading that opens with an ID, and its title.
var definingHeading = regexp.MustCompile(`^#{1,4}\s+(?:\*\*)?(` + idText + `)(?:\*\*)?(?:\s*[—:–-]\s*|\s+|$)(.*)$`)

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
		r := row{feature: feature, file: file, subject: subjectOf(cells[1:])}
		if s := samID.FindStringSubmatch(cells[0]); s != nil {
			r.family, r.num = samFamily, s[1]
		} else {
			r.family, r.num, r.suffix = idset.Parts(idset.ID.FindStringSubmatch(cells[0]))
		}
		out = append(out, r)
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

// releases is each feature folder's release, where the record states one: in
// the folder's name, else in its project brief's title.
func releases(docs string) map[string]string {
	out := map[string]string{}
	entries, _ := os.ReadDir(filepath.Join(docs, "02_features"))
	for _, e := range entries { // bounded by the features (P10-02)
		if v := idset.Release.FindString(e.Name()); v != "" && strings.HasPrefix(e.Name(), v) {
			out[e.Name()] = v
			continue
		}
		b, err := os.ReadFile(filepath.Join(docs, "02_features", e.Name(), "01-objectives", "project-brief.md"))
		if err != nil {
			continue
		}
		for _, line := range strings.SplitN(string(b), "\n", 12) { // the front matter (P10-02)
			if strings.HasPrefix(line, "title:") {
				out[e.Name()] = idset.Release.FindString(line)
			}
		}
	}
	return out
}

// render writes the index: a section per family, in idset's order, and in
// each the IDs by number, every feature's row under its ID.
func render(rows []row, rel map[string]string) string {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if oa, ob := familyOrder(a.family), familyOrder(b.family); oa != ob {
			return oa < ob
		}
		if a.family != b.family {
			return a.family < b.family // two UAT rounds
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
		release := rel[r.feature]
		if r.family == sfFamily {
			release = sfRelease // the finding's release, not the one that mentions it
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n", written(r), release, where, strings.ReplaceAll(r.subject, "|", "\\|"))
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
file's ` + "`git log --follow`" + ` says which release wrote the comment, across a move. **A new citation names its release**
(` + "`0.19.5 D-4`" + `, or a release's own prefix such as ` + "`MVS-D-80`" + `), which ` + "`tools/authoring`" + ` holds
over the lines a branch adds (` + "`AP-CITE-01`" + `). A row defines an ID when the ID is the whole of its first
cell; a later table that mentions it is not listed. Where no table defines an ID, its row is its
best mention, marked "Mentioned:", or the MVS-D section below: 0.16.0's later rulings and the
follow-ups only ever mentioned (0.19.5 D-37, D-38). A closed follow-up the code cites resolves to
` + "`06_docs/follow-ups-closed.md`" + `, its row as it was when it closed.

A UAT round's findings (U1-, U2-) and a plan's tasks (W1.17, task 17 of wave 1; a wave alone is no ID)
are indexed as every other family is. SAM-D, the severe-alerts release's rulings, carries its release
in its prefix as MVS-D does; S-F, 0.9.0's red team's findings, from before the record, is indexed at
each one's best mention (0.19.6 D-50).
`
