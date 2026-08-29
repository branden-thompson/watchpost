// Package pronounce is the library of pronunciation rules — how the
// correspondent reads abbreviations, zones, states and product contractions —
// kept as editable tables, not as maps in the code (HUM LEAD 2026-08-28: the
// same shape as the report scripts; a rule is a line in a file).
//
// Convention — one file per table:
//
//	rules/<table>.txt   "KEY<TAB>spoken form" per line for a table;
//	                    "KEY" per line for a set; '#' comments; blank lines ignored
//
// The tables today: zones (time zones, every script), abbreviations (NWS
// product contractions), states (postal codes as names), states-ambiguous
// (the codes that are also words — a set), words (voice-only spellings).
// The synth's normaliser and its voice-only pass load them by name; adding
// a rule is adding a line, adding a table is adding a file. The tables are
// compiled in — Go has no hot swap — so this is for the maintainer's hands
// and eyes, not the runtime.
package pronounce

import (
	"embed"
	"io/fs"
	"sort"
	"strings"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

//go:embed rules/*.txt
var builtin embed.FS

const root = "rules"

// Table returns a rule table by name: KEY → spoken form, parsed on each call
// (microseconds; a memo would be a package global, which P10-06 forbids —
// red-team round 4, A-11 declined on that ground). An unknown table is empty
// (the tests pin the names the code asks for). A duplicated key is a
// maintainer's slip: the first rule wins and the table is still served.
func Table(name string) map[string]string {
	out := map[string]string{}
	for _, line := range lines(name) {
		k, v, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if _, dup := out[k]; invariant.Check(!dup && k != "" && v != "", "a rule is one key, one spoken form, once") != nil {
			continue
		}
		out[k] = v
	}
	return out
}

// Set returns a rule set by name: the keys listed, one per line (a copy).
func Set(name string) map[string]bool {
	out := map[string]bool{}
	for _, line := range lines(name) {
		k, _, _ := strings.Cut(line, "\t")
		out[strings.TrimSpace(k)] = true
	}
	return out
}

// Names lists the tables on file, sorted.
func Names() []string {
	var out []string
	entries, _ := fs.ReadDir(builtin, root)
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".txt"); ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// lines reads a table's rule lines: comments and blanks dropped. One pass
// over the file's lines (bounded by their count — P10-02).
func lines(name string) []string {
	if err := invariant.Check(!strings.ContainsAny(name, "/\\."), "a table name never leaves the rules directory"); err != nil {
		return nil
	}
	b, err := fs.ReadFile(builtin, root+"/"+name+".txt")
	if err != nil {
		return nil
	}
	raw := strings.Split(string(b), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimRight(line, " \r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}
