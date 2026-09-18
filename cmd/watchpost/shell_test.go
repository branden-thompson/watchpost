package main

// shell_test.go — THE SHELL LEDGER. Everything in this repository is Go unless
// absolutely necessary (HUM LEAD, 2026-09-17), and "necessary" is a ruling: a
// non-Go source file under scripts/ exists only with a ratified row here saying
// why, and each row is a port candidate (F-156). AP-SHELL-01 in tools/authoring
// keeps shell out of Go files; this keeps new shell files out of the tree.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// shellScripts are the non-Go sources under scripts/, each with the reason it
// is still shell. A file with no row fails TestEveryShellScriptHasALedgerRow; a
// row whose file is gone fails the registry.
var shellScripts = exempt(&exemptionTable{
	name: "shellScripts", satisfied: "tools/authoring/main.go",
	rows: map[string]string{
		"scripts/install-test.sh":                   "installs and runs a built artifact on a clean CI runner; port candidate F-156",
		"scripts/install.sh":                        "the published curl-to-sh installer; a user-facing contract, port candidate F-156",
		"scripts/lint-imports.sh":                   "grep over import paths; port candidate F-156 (tools/authoring already walks every file)",
		"scripts/lint-injector.sh":                  "strings over a built binary; port candidate F-156",
		"scripts/lint-watermark.sh":                 "grep for attribution lines; port candidate F-156 (tools/authoring already walks every file)",
		"scripts/lint.sh":                           "wraps golangci-lint with a baseline ratchet; F-118 names its fail-open defect, port candidate F-156",
		"scripts/sync-go-studs.sh":                  "the go-studs patch stack; port candidate F-156",
		"scripts/third-party-licenses.sh":           "collects module licences; port candidate F-156",
		"scripts/quality/dump.sh":                   "a debugging dump helper; delete or port, F-156",
		"scripts/quality/exposure-scan.py":          "a one-off exposure scan; delete or port, F-156",
		"scripts/quality/ledger-ratified.sh":        "checks P10 ledger ratification; port candidate F-156",
		"scripts/quality/lint-ledger.sh":            "checks the P10 ledger mirror; port candidate F-156",
		"scripts/quality/lookup-stall-probe.expect": "drives the TUI under expect; expect has no Go equivalent in the tree yet, F-156",
		"scripts/quality/mutant-anchors.sh":         "checks mutant anchors still resolve; port candidate F-156",
		"scripts/quality/mutant-verdicts.sh":        "runs the mutant sweep and records verdicts; port candidate F-156",
		"scripts/quality/p10-ledger-mirror.py":      "generates the P10 ledger mirror; port candidate F-156",
		"scripts/quality/p10-unmatched.sh":          "checks P10 findings against the ledger; port candidate F-156",
		"scripts/quality/p10-unmatched_test.sh":     "the sibling control of p10-unmatched.sh; ports with it, F-156",
		"scripts/quality/severe-modal.expect":       "drives the TUI under expect; expect has no Go equivalent in the tree yet, F-156",
		"scripts/quality/soak-1h.expect":            "drives the TUI under expect; expect has no Go equivalent in the tree yet, F-156",
		"scripts/quality/soak-phases.expect":        "drives the TUI under expect; expect has no Go equivalent in the tree yet, F-156",
		"scripts/quality/soak.sh":                   "the soak harness around the expect scripts; ports with them, F-156",
		"scripts/quality/validate-journey.expect":   "drives the TUI under expect; expect has no Go equivalent in the tree yet, F-156",
		"06_docs/mutants/run.sh":                    "runs the mutant corpus by hand outside make; port candidate F-156",
		"tools/geotrim/refresh.sh":                  "refreshes geotrim's fixture data; port candidate F-156",
	},
	exists:      func(t *testing.T, p string) bool { _, err := os.Stat(filepath.Join("../..", p)); return err == nil },
	stillNeeded: func(t *testing.T, p string) bool { return isNonGoSource(filepath.Join("../..", p)) },
})

// isNonGoSource: a regular file that is not Go and not data, and is either
// under scripts/ or an executable with a shebang anywhere in the tree.
func isNonGoSource(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".txt") {
		return false
	}
	if strings.Contains(filepath.ToSlash(path), "/scripts/") {
		return true
	}
	if info.Mode()&0o111 == 0 {
		return false
	}
	head := make([]byte, 2)
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	n, _ := f.Read(head)
	_ = f.Close() // read only; nothing to lose on close
	return n == 2 && string(head) == "#!"
}

// EVERY NON-GO SOURCE — under scripts/, or any tracked executable with a
// shebang — HAS A RATIFIED ROW. Derived from git's index, never from a list.
func TestEveryShellScriptHasALedgerRow(t *testing.T) {
	out, err := exec.Command("git", "-C", "../..", "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("COULD NOT RUN — git ls-files: %v", err)
	}
	var seen int
	for _, rel := range strings.Split(string(out), "\x00") { // bounded by the index (P10-02)
		if rel == "" || strings.HasPrefix(rel, "third_party/") || !isNonGoSource(filepath.Join("../..", rel)) {
			continue
		}
		seen++
		if _, ok := shellScripts[rel]; !ok {
			t.Errorf("%s is a non-Go source with no row in the shell ledger. Everything is Go unless absolutely "+
				"necessary; a new script needs a ruling, and the ruling is a row here.", rel)
		}
	}
	if seen == 0 {
		t.Fatalf("COULD NOT RUN — no non-Go source found in the index")
	}
}
