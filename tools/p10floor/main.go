// Command p10floor holds each P10-05 density row to the figure it was
// ratified at (D-264): a row whose reason states "FLOOR x.xx" fails when the
// package it covers measures under that figure, so a row ratified over one
// package cannot quietly cover a thinner one. It reads the JSON `a2dh p10
// check --json` writes, where each exempted finding carries its row's reason.
// A firing row that states no floor fails too, named with the figure it
// measures, so its floor is written at that figure (D-277).
//
//	go run ./tools/p10floor P10_JSON
//	go run ./tools/p10floor --self-test
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// densityRule is the P10 rule whose rows carry a floor.
const densityRule = "P10-05-INVARIANT-DENSITY"

// measured is a density finding's figure; floor is a row's ratified one.
var (
	measured = regexp.MustCompile(`invariant density ([0-9]+\.[0-9]+)`)
	floor    = regexp.MustCompile(`FLOOR ([0-9]+\.[0-9]+)`)
)

// finding is the part of a2dh's finding this reads.
type finding struct {
	File         string `json:"file"`
	RuleID       string `json:"rule_id"`
	Message      string `json:"message"`
	Exempted     bool   `json:"exempted"`
	ExemptReason string `json:"exempt_reason"`
}

var errUsage = errors.New("usage: p10floor P10_JSON | --self-test")

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errOut io.Writer) int {
	fail := func(code int, words ...any) int {
		_, _ = fmt.Fprintln(errOut, words...)
		return code
	}
	if len(args) != 1 {
		return fail(2, errUsage)
	}
	if args[0] == "--self-test" {
		if err := selfTest(); err != nil {
			return fail(1, "p10floor SELF-TEST FAILED:", err)
		}
		_, _ = fmt.Fprintln(out, "p10floor self-test: a row under its floor refused, one at it passed, an unread figure refused (OK)")
		return 0
	}
	b, err := os.ReadFile(args[0])
	if err != nil {
		return fail(2, "p10floor:", err)
	}
	var report struct {
		Findings []finding `json:"findings"`
	}
	err = json.Unmarshal(b, &report)
	if err != nil {
		return fail(2, "p10floor: the P10 report is not JSON:", err)
	}
	held, under, floorless, err := judge(report.Findings)
	if err != nil {
		return fail(2, "p10floor:", err)
	}
	if len(under) > 0 {
		return fail(1, fmt.Sprintf("p10floor: %d row(s) under the floor they were ratified at (D-264) - add real checks, or put the row to the HUM LEAD again:\n  %s", len(under), strings.Join(under, "\n  ")))
	}
	if len(floorless) > 0 {
		return fail(1, fmt.Sprintf("p10floor: %d firing density row(s) state no floor (D-277) - write each row's floor at the figure below, \"FLOOR x.xx over n changed functions\", as an A-row for the HUM LEAD's veto:\n  %s", len(floorless), strings.Join(floorless, "\n  ")))
	}
	_, _ = fmt.Fprintf(out, "p10floor: %d density row(s) at or over their floors\n", held)
	return 0
}

// judge counts the exempted density findings whose rows state a floor,
// names each measuring under it, and names each whose row states none with
// what it measures; a row's floor or a finding's figure that cannot be read
// is an error, never a pass.
func judge(findings []finding) (held int, under, floorless []string, err error) {
	for _, f := range findings {
		if f.RuleID != densityRule || !f.Exempted {
			continue // a live finding fails the P10 check itself
		}
		mm := measured.FindStringSubmatch(f.Message)
		if mm == nil {
			return 0, nil, nil, fmt.Errorf("%s: the finding states no density: %q", f.File, f.Message)
		}
		fm := floor.FindStringSubmatch(f.ExemptReason)
		if fm == nil {
			floorless = append(floorless, fmt.Sprintf("%s: measures %s%s, its row states no floor", f.File, mm[1], over(f.Message)))
			continue
		}
		want, werr := strconv.ParseFloat(fm[1], 64)
		got, gerr := strconv.ParseFloat(mm[1], 64)
		if werr != nil || gerr != nil {
			return 0, nil, nil, fmt.Errorf("%s: a figure that is no number: floor %q, measured %q", f.File, fm[1], mm[1])
		}
		held++
		if got < want {
			under = append(under, fmt.Sprintf("%s: %s, under its floor %s", f.File, mm[1], fm[1]))
		}
	}
	return held, under, floorless, nil
}

// functions is a density finding's count of changed functions.
var functions = regexp.MustCompile(`across ([0-9]+) changed function`)

// over is " over n changed functions" from a finding's message, or nothing
// when it does not say.
func over(message string) string {
	m := functions.FindStringSubmatch(message)
	if m == nil {
		return ""
	}
	return " over " + m[1] + " changed functions"
}

// selfTest proves the instrument can fail in this tree: a planted row under
// its floor is refused, one at its floor passes, one with no floor is named,
// and a finding whose figure does not read is an error.
func selfTest() error {
	row := func(density, reason string) finding {
		return finding{File: "planted/p.go", RuleID: densityRule, Message: "P10-05: invariant density " + density + " < 2.0", Exempted: true, ExemptReason: reason}
	}
	held, under, floorless, err := judge([]finding{row("0.42", "FLOOR 0.43 over 9"), row("0.43", "FLOOR 0.43 over 9"), row("0.10", "ratified by precedent")})
	if err != nil {
		return err
	}
	if held != 2 || len(under) != 1 || len(floorless) != 1 {
		return fmt.Errorf("three planted rows: %d held, %d under, %d with no floor; want 2, 1 and 1", held, len(under), len(floorless))
	}
	_, _, _, err = judge([]finding{{File: "planted/p.go", RuleID: densityRule, Message: "no figure", Exempted: true, ExemptReason: "FLOOR 0.43"}})
	if err == nil {
		return errors.New("a finding with no figure passed")
	}
	return nil
}
