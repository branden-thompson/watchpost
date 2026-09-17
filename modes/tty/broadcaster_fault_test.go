package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// F-150, REVIEW — THE OPERATOR SEES A FAULT RUN ON THE CONSOLE, in the station's
// own band while ON AIR: the count and the reason. A read that finished clears
// it (Run 0), and so does STANDBY, which is the operator acting on it.
func TestAFaultRunShowsInTheBandWhileOnAirAndClears(t *testing.T) {
	b := bcStandby(t, true, 0)
	b, _ = b.Update(StationMsg{Power: lineup.Running})
	b, _ = b.Update(StationFaultMsg{Run: 3, Reason: "the report could not be composed: no key"})
	got := stripANSITest(b.View().Content)
	for _, want := range []string{"3 CARD(S) FAILED", "could not perform", "no key"} { // bounded by the phrases (P10-02)
		if !strings.Contains(got, want) {
			t.Errorf("an ON AIR station with three failed cards shows no %q:\n%s", want, got)
		}
	}
	cleared, _ := b.Update(StationFaultMsg{})
	if strings.Contains(stripANSITest(cleared.View().Content), "FAILED") {
		t.Error("a read that finished did not clear the fault band")
	}
	standby, _ := b.Update(StationMsg{Power: lineup.OffAir})
	if strings.Contains(stripANSITest(standby.View().Content), "FAILED") {
		t.Error("STANDBY did not clear the fault band")
	}
	// AND IT STAYS CLEARED ON THE WAY BACK UP (R2 review F4): the band hides
	// itself off the air, so the only way to see whether standby CLEARED the
	// fault or merely covered it is to go back ON AIR.
	back, _ := standby.Update(StationMsg{Power: lineup.Running})
	if strings.Contains(stripANSITest(back.View().Content), "FAILED") {
		t.Error("the fault band came back ON AIR after standby — standby covered it rather than clearing it")
	}
}

// REVIEW 2026-09-17 (ruling 6-ii) — THE LISTENER'S MUTE IS SHOWN ON AIR. [M] in
// the Observer declines every hazard read on a station the operator put ON AIR,
// and the console said nothing. The Router hands the Observer's mute to the
// console, which shows it in the station's own band while ON AIR.
func TestAMutedListenerIsShownWhileTheStationIsOnAir(t *testing.T) {
	b := bcStandby(t, true, 0)
	b, _ = b.Update(StationMsg{Power: lineup.Running})
	b.listenerMuted = true
	if got := stripANSITest(b.View().Content); !strings.Contains(got, "LISTENER MUTED") {
		t.Errorf("an ON AIR station with the listener muted shows no LISTENER MUTED:\n%s", got)
	}
	b.listenerMuted = false
	if got := stripANSITest(b.View().Content); strings.Contains(got, "LISTENER MUTED") {
		t.Error("the mute band persists after the mute is lifted")
	}
	r := routerAt(lineup.Running, SurfaceBroadcaster)
	r.observer.tickerMuted = true
	m, _ := r.Update(StationMsg{Power: lineup.Running})
	if !m.(Router).broadcaster.listenerMuted {
		t.Error("the Router did not hand the Observer's mute to the console")
	}
}

// REVIEW 2026-09-17 (ruling 8) — THE CONSOLE SAYS HOW MANY PLACES ARE IN REACH
// AT THE STATION'S RADIUS. Measured around the real index, a three-mile station
// has a pool of ONE place; a console that only lists it reads like a fifty-mile
// station that is slow. The pool's footer states the count and the radius, and
// an empty pool says so rather than going quiet.
func TestThePoolFooterSaysHowManyPlacesAreInReach(t *testing.T) {
	b := bcStandby(t, false, 0)
	b, _ = b.Update(StationMsg{Power: lineup.Running})
	b.area.Pool = []snapshot.LocationRef{{Label: "Bonsall, CA", Lat: 33.2889, Lon: -117.2153}}
	b.area.RadiusMi = 3
	b.areaGen++ // the memo keys on the area generation, which the real area message bumps
	if got := stripANSITest(b.View().Content); !strings.Contains(got, "1 in reach at 3 mi") {
		t.Errorf("a three-mile station with one place shows no reach line:\n%s", got)
	}
	b.area.Pool = nil
	b.areaGen++
	if got := stripANSITest(b.View().Content); !strings.Contains(got, "0 places in reach at 3 mi") {
		t.Errorf("an empty pool says nothing about its reach:\n%s", got)
	}
}
