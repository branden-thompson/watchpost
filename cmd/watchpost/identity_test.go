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
	"testing"

	"github.com/branden-thompson/watchpost/tools/internaltrees/trees"
)

// identityPatterns are the classes that must never reach a public tree.
//
// EACH CARRIES WHAT IT COSTS A READER, because a gate that only says "refused"
// teaches nobody why. These are the same expressions lint-ledger.sh applies to
// the ledger mirror, plus the harness-scratchpad class it does not know about —
// which was the larger of the two leaks.
var identityPatterns = []struct {
	name string
	re   *regexp.Regexp
	why  string
}{
	{"an absolute home directory", regexp.MustCompile(`(^|[^A-Za-z0-9._-])/(Users|home)/[a-z][a-z0-9._-]{2,}`),
		"it names a person's account and a layout nobody outside can use"},
	{"a home-relative personal path", regexp.MustCompile("(^|[\\s\"'`(])~/Desktop/|~/[A-Za-z0-9._-]*PERSONAL"),
		"it names one person's desktop layout, which tells a public reader nothing"},
	{"an agent-harness scratchpad path", regexp.MustCompile(`/private/tmp/claude-[0-9]+/`),
		"it names the tooling a human used and the session they used it in"},
	{"an internal project tree", internalTrees,
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
		for _, pat := range identityPatterns { // bounded by the pattern list (P10-02)
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
		for _, pat := range identityPatterns { // bounded by the pattern list (P10-02)
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
// A rule that cannot be built stops the package rather than scanning without it:
// a gate missing one class still prints a pass.
var internalTrees = func() *regexp.Regexp {
	expr, err := trees.Expr(filepath.Join("..", ".."), os.Getenv("HOME"))
	if err != nil {
		panic("identity gate: cannot build the internal-tree rule: " + err.Error())
	}
	return regexp.MustCompile(expr)
}()

// ONE DEFINITION, OR THE NEXT RENAME FINDS A COPY. Every gate that is not Go must
// ask tools/internaltrees; a private copy of the old two-name pattern in any of
// them is the defect package trees was written to remove.
func TestEveryIdentityGateReadsTheOneTreeRule(t *testing.T) {
	oldCopy := "LI_PROJECTS" + "|" + "DESIGN_FOUNDATIONS"
	for _, rel := range []string{"scripts/quality/lint-ledger.sh", "scripts/quality/p10-ledger-mirror.py"} { // bounded (P10-02)
		body, err := os.ReadFile(filepath.Join("..", "..", rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		if !strings.Contains(string(body), "tools/internaltrees") {
			t.Errorf("%s does not read the rule from tools/internaltrees", rel)
		}
		if strings.Contains(string(body), oldCopy) {
			t.Errorf("%s still carries its own copy of the internal-tree pattern", rel)
		}
	}
}
