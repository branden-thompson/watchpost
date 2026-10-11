package app

import (
	"strings"
	"testing"
)

// creditPending are the propagation library's sources About does not credit
// yet, each with why: the day one is read or reproduced, it is credited and
// leaves this list.
var creditPending = map[string]string{
	"GIRO":        "not read until go-ionomaps' station assimilation (G5, A-35)",
	"ITU-R P.533": "cited for its method; no text or tables reproduced",
	"P.1239":      "cited for its method; no text or tables reproduced",
}

// TestEveryPropagationSourceIsCredited is FR-4.1: every source the
// propagation library names is in About's credits - by the host it asks, or
// by its badge where it is read at build time - unless it is pending, and a
// pending one is not credited, so the list stays true.
func TestEveryPropagationSourceIsCredited(t *testing.T) {
	hosts, badges := map[string]bool{}, map[string]bool{}
	for _, g := range creditGroups() {
		for _, l := range g.Lines {
			hosts[l.Host], badges[l.Badge] = true, true
		}
	}
	sources := propagationSources()
	if len(sources) == 0 {
		t.Fatal("the library names no sources: this measures nothing")
	}
	for _, s := range sources {
		credited := badges[strings.ToUpper(s.Name)]
		for _, h := range s.Hosts {
			credited = credited || hosts[h]
		}
		why, pending := creditPending[s.Name]
		switch {
		case !credited && !pending:
			t.Errorf("%s (%v) is not in About's credits", s.Name, s.Hosts)
		case credited && pending:
			t.Errorf("%s is credited and still listed as pending (%s)", s.Name, why)
		}
	}
}
