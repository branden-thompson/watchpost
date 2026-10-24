package app

import (
	"strings"
	"testing"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// creditPending are the propagation library's sources About does not credit
// yet, each with why: the day one is read or reproduced, it is credited and
// leaves this list.
var creditPending = map[string]string{
	"ITU-R P.533": "cited for its method; its tables reach the library through ITUHFProp, credited as such (D-186)",
	"P.1239":      "cited for its method; no text or tables reproduced",
	"P.371":       "cited for its method, the sunspot number from the flux by its equation; no text or tables reproduced",
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
	sources := ionomaps.Sources()
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

// TestEveryStatusSourceIsGroupedAndCredited is D-170 (A-43): every station
// provider, every map source and every named host serves a feature and has
// a badge, and every badge is About's chip for a source - or a source About
// does not credit yet (creditPending), so the Status window never names a
// host by a chip the app does not show.
func TestEveryStatusSourceIsGroupedAndCredited(t *testing.T) {
	badges := map[string]bool{}
	for _, g := range creditGroups() {
		for _, l := range g.Lines {
			badges[l.Badge] = true
		}
	}
	known := func(badge string) bool { return badges[badge] || creditPending[badge] != "" }
	sources := providerSources()
	for id := range providerEndpoints() {
		s, ok := sources[id]
		if !ok || s.Feature == 0 || !known(s.Badge) {
			t.Errorf("provider %s: %+v; want a feature and a credited badge", id, s)
		}
	}
	for host, s := range hostSources() {
		if s.Feature == 0 || !known(s.Badge) {
			t.Errorf("host %s: %+v; want a feature and a credited badge", host, s)
		}
	}
	for _, s := range mapSourceList() {
		if !known(s.Badge) {
			t.Errorf("map source %s (%s) has the badge %q, which About does not show", s.Name, s.Host, s.Badge)
		}
	}
}
