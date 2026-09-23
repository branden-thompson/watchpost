package main

// identity_test.go — the published tree names no person and no machine.
//
// THIS REPOSITORY IS PUBLIC, and the class has now leaked twice. A 0.14.0 sweep
// deleted 32 pprof profiles for embedding a home directory and did not gate the
// class, so it returned in a text profile, nine benchmark overlays and an
// archived spike script — 22 files across two pushed branches, found by four
// independent reviewers on the same day.
//
// THE RULE EXISTED AND WAS SCOPED TO ONE FILE. `scripts/quality/lint-ledger.sh`
// refuses exactly these patterns, and refuses them only in the P10 ledger mirror
// — then prints "no machine paths, no harness paths". A rule applied to one file
// while its class is live in twenty-one others is the shape this project calls a
// verifier that cannot verify.
//
// SO THIS ASKS THE WHOLE INDEX. It is deliberately a different question from
// `scripts/quality/exposure-scan.py`, which surveys and reports and whose own
// header says it is "a survey, not a gate": a survey nobody runs is what let the
// count sit published and unacted-on.

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/branden-thompson/watchpost/tools/internaltrees/trees"
)

// identityPatterns are the classes that must never reach a public tree.
//
// EACH CARRIES WHAT IT COSTS A READER, because a gate that only says "refused"
// teaches nobody why. These are the same expressions lint-ledger.sh applies to
// the ledger mirror, plus the harness-scratchpad class it does not know about —
// which was the larger of the two leaks.
type identityPattern struct {
	name string
	re   *regexp.Regexp
	why  string
}

var identityPatterns = []identityPattern{
	{"an absolute home directory", regexp.MustCompile(`(^|[^A-Za-z0-9._-])/(Users|home)/[a-z][a-z0-9._-]{2,}`),
		"it names a person's account and a layout nobody outside can use"},
	{"a home-relative personal path", regexp.MustCompile("(^|[\\s\"'`(])~/Desktop/|~/[A-Za-z0-9._-]*PERSONAL"),
		"it names one person's desktop layout, which tells a public reader nothing"},
	{"an agent-harness scratchpad path", regexp.MustCompile(`/private/tmp/claude-[0-9]+/`),
		"it names the tooling a human used and the session they used it in"},
	// nil until the test that scans builds it: see internalTrees below.
	{"an internal project tree", nil,
		"it says where internal tooling lives, which tells a public reader nothing"},
	{"an email address", regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`),
		"it is personal data"},
	// RFC 2606 / 6761 RESERVE example.com, .invalid, .test and .localhost so that
	// documentation and fixtures can name an address without naming a person. A
	// gate that flagged them would be telling an author to replace the correct
	// value with a worse one, which is the failure mode worse than the defect.
}

// identityExempt are tracked files allowed to carry a match, with the reason.
//
// THE ONLY LEGITIMATE CASES ARE THE RULES THEMSELVES, and a fixture that must
// contain the thing it refuses. A row here is a file a reviewer has agreed may
// name one of these; it is not a place to park a leak.
var identityExempt = exempt(&exemptionTable{
	name: "identityExempt", satisfied: "go.mod",
	rows: map[string]string{
		"cmd/watchpost/identity_test.go":                                       "this file — the patterns and their exemptions have to be written down somewhere",
		"scripts/quality/lint-ledger.sh":                                       "the ledger linter's own rules and its self-test probes, which must contain what they refuse",
		"tools/internaltrees/trees/trees.go":                                   "the internal-tree rule itself — its retired names are the patterns it refuses",
		"tools/internaltrees/trees/trees_test.go":                              "the internal-tree rule's controls, which must contain what they refuse",
		"THIRD_PARTY_LICENSES.md":                                              "upstream authorship, reproduced because the licences require it; the addresses are the copyright holders' own",
		"06_docs/02_features/severe-alerts-modals/04-development/p1-domain.md": "a public NOAA office contact, quoted as domain research — it is published by the agency and names no one here",
		"domains/globalfeed/testdata/nws_active_unfiltered_trimmed.json":       "a captured NWS payload; the webmaster address in it is the agency's own, and rewriting a fixture forges it",
		"domains/radio/stream/testdata/weatherusa.json":                        "a captured WeatherUSA payload; the support address in it is the service's own, and rewriting a fixture forges it",
	},
	exists: fileExists,
	stillNeeded: func(t *testing.T, rel string) bool {
		body, err := os.ReadFile(filepath.Join("..", "..", rel))
		if err != nil {
			return false
		}
		for _, pat := range identityRules(t) { // bounded by the pattern list (P10-02)
			if m := pat.re.FindString(string(body)); m != "" && !reservedForDocs.MatchString(m) {
				return true
			}
		}
		return false
	},
})

// reservedForDocs are the names RFC 2606 and 6761 set aside for examples. A
// match on one of these is a fixture doing the right thing.
var reservedForDocs = regexp.MustCompile(`(?i)@(example\.(com|org|net)|[a-z0-9.-]*\.(invalid|test|localhost|example))$|/(Users|home)/(user|you|someone|me|<[a-z]+>)\b`)

// identity-gate: scans the index for every class.
func TestThePublishedTreeNamesNoPersonOrMachine(t *testing.T) {
	root := filepath.Join("..", "..")
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v — this check measures nothing without it", err)
	}
	paths := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	if len(paths) < 100 {
		t.Fatalf("the index holds %d files; this check has lost its subject", len(paths))
	}

	rules := identityRules(t) // once: the same table for every file
	var scanned int
	for _, p := range paths { // bounded by the index (P10-02)
		// A FIXTURE IS EXEMPTED BY NAME, NOT BY DIRECTORY. A captured upstream
		// payload under testdata/ keeps the addresses the service sent (rewriting
		// one would forge the thing it reproduces), but each such file is a row
		// in identityExempt with that reason — a blanket testdata/ skip would
		// let the next fixture carry anything (REVIEW 2026-09-17).
		if p == "" || identityExempt[p] != "" || isBinaryPath(p) {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			continue
		}
		scanned++
		for _, pat := range rules { // bounded by the pattern list (P10-02)
			m := pat.re.FindString(string(body))
			if m == "" || reservedForDocs.MatchString(m) {
				continue
			}
			t.Errorf("%s carries %s (%q) and is TRACKED in a public repository.\n"+
				"Why it matters: %s.\n"+
				"Remove the artefact if it is machine output nobody cites, or replace the value with a "+
				"placeholder if the file is a record that needs to keep its meaning. Do not add an "+
				"exemption row unless the file's purpose is to state the rule itself.", p, pat.name, m, pat.why)
			break
		}
	}
	if scanned < 100 {
		t.Fatalf("scanned %d text files; this check has lost its subject", scanned)
	}
}

// isBinaryPath skips the assets a text scan cannot read meaningfully. It is a
// skip for THIS check only — a tracked executable is TestNoTrackedBinaries's
// question, and that gate runs beside this one.
func isBinaryPath(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".png", ".gif", ".jpg", ".jpeg", ".webp", ".gz", ".zip", ".pb", ".ico", ".woff", ".woff2":
		return true
	}
	return false
}

// internalTrees is the internal-project-tree class from package trees — the one
// definition lint-ledger.sh and p10-ledger-mirror.py read too, through
// tools/internaltrees. Its own controls live beside it in trees_test.go.
//
// A rule that cannot be built FAILS THE TEST THAT NEEDS IT rather than the
// package: a gate missing one class still prints a pass, so it must not be
// skipped - but it was built in a package-level initializer, where a plain
// permissions error on the workspace took down every test in cmd/watchpost,
// most of which have nothing to do with identity (red team, DISCOVER exit
// 2026-09-22). Built once, on first use, by the test that reads it.
var internalTrees = sync.OnceValues(func() (*regexp.Regexp, error) {
	expr, err := trees.Expr(filepath.Join("..", ".."), os.Getenv("HOME"))
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}
	return re, nil
})

// identityRules is the pattern table with the internal-tree rule filled in. A
// rule that cannot be built fails the caller, loudly, with the reason - never a
// scan that quietly covers one class fewer.
func identityRules(t *testing.T) []identityPattern {
	t.Helper()
	re, err := internalTrees()
	if err != nil {
		t.Fatalf("COULD NOT RUN — cannot build the internal-tree rule, so one class would go unscanned: %v", err)
	}
	out := make([]identityPattern, 0, len(identityPatterns))
	var filled int
	for _, pat := range identityPatterns { // bounded by the pattern list (P10-02)
		if pat.re == nil {
			pat.re, filled = re, filled+1
		}
		out = append(out, pat)
	}
	// EXACTLY ONE ROW IS FILLED IN. A nil is a positional sentinel, and a
	// second one - a typo, a row added without its expression - would silently
	// be scanned with the tree rule and its own class would go unchecked.
	if filled != 1 {
		t.Fatalf("COULD NOT RUN — %d pattern rows have no expression; exactly one (the internal-tree row) may", filled)
	}
	return out
}

// ONE DEFINITION, OR THE NEXT RENAME FINDS A COPY. Every gate that is not Go must
// ask tools/internaltrees; a private copy of the old two-name pattern in any of
// them is the defect package trees was written to remove.
// IT ASSERTS AN EXECUTION, NOT A MENTION. The first version of this test was
// satisfied by the string `tools/internaltrees` appearing anywhere - including
// in the comment that sits directly above the live call - so deleting the call
// and keeping the comment left the test green with the rule gone (red team,
// DISCOVER exit 2026-09-22). Comment lines are stripped before the check.
// identity-gate: every non-Go gate asks for the one rule.
func TestEveryIdentityGateReadsTheOneTreeRule(t *testing.T) {
	// The literal is safe here: this file is its own exemption row.
	oldCopy := "LI_PROJECTS|DESIGN_FOUNDATIONS"
	for _, rel := range []string{"scripts/quality/lint-ledger.sh", "scripts/quality/p10-ledger-mirror.py"} { // bounded (P10-02)
		body, err := os.ReadFile(filepath.Join("..", "..", rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		code := withoutComments(string(body))
		if !strings.Contains(code, "./tools/internaltrees") {
			t.Errorf("%s does not RUN the rule from tools/internaltrees (a mention in a comment is not a call)", rel)
		}
		if strings.Contains(code, oldCopy) {
			t.Errorf("%s still carries its own copy of the internal-tree pattern", rel)
		}
	}
}

// withoutComments drops every form of comment the two consumers can carry, so
// a mention cannot stand in for a call: a whole-line `#`, a `#` after code on
// the same line, and the triple-quoted blocks the Python consumer documents
// itself with.
//
// IT ERRS TOWARDS STRIPPING. A `#` inside a shell string is cut here too, so a
// call that appears only inside such a string is not counted — the check gets
// stricter, never looser, which is the safe direction for a guard asking
// whether a rule is really read.
func withoutComments(body string) string {
	body = tripleQuoted.ReplaceAllString(body, "")
	out := make([]string, 0, 64)
	for _, line := range strings.Split(body, "\n") { // bounded by the file (P10-02)
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// tripleQuote is the docstring delimiter, built rather than written so this
// file's own text carries no triple quote.
var tripleQuote = strings.Repeat(`"`, 3)

// tripleQuoted matches a Python docstring in either quote style.
var tripleQuoted = regexp.MustCompile(`(?s)""".*?"""|'''.*?'''`)

// TestTheOneTreeRuleCheckSeesThroughAComment is the positive control for the test
// above: the commented-out form must FAIL the check, and the live form must
// pass it. Without this control, a `withoutComments` that stopped stripping
// would leave the original hole open and nothing would say so.
// identity-gate: the control for the check above.
func TestTheOneTreeRuleCheckSeesThroughAComment(t *testing.T) {
	commented := "# TREES=$(go run ./tools/internaltrees)\nexit 0\n"
	if strings.Contains(withoutComments(commented), "./tools/internaltrees") {
		t.Error("a commented-out call counted as a call")
	}
	live := "# reads the rule from ./tools/internaltrees\nTREES=$(go run ./tools/internaltrees)\n"
	if !strings.Contains(withoutComments(live), "./tools/internaltrees") {
		t.Error("a live call was stripped")
	}
	trailing := "exit 0  # see ./tools/internaltrees\n"
	if strings.Contains(withoutComments(trailing), "./tools/internaltrees") {
		t.Error("a mention after code on the same line counted as a call")
	}
	docstring := tripleQuote + "reads the rule from ./tools/internaltrees." + tripleQuote + "\nexit 0\n"
	if strings.Contains(withoutComments(docstring), "./tools/internaltrees") {
		t.Error("a mention inside a docstring counted as a call")
	}
}
