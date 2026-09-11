package app

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/platform/config"
)

func bedPipelines(t *testing.T) *livePipelines {
	t.Helper()
	tbl, err := stream.LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{idx: indexForTest(t), relayTable: tbl}
	lp.setStation(stationFrom(config.Config{Locations: []config.Location{bonsallCfg}}))
	lp.bedRadiusMi = config.DefaultBedRadiusMi
	return lp
}

// THE SELECTOR WALKS THE STATION'S FENCE, NOT THE LISTENER'S WATCHLIST (D-78).
//
//	"Lone Pine's 'Fresno' relay is completely inappropriate for being the relay"
//	 — HUM LEAD, 2026-09-10
//
// WHAT IT WALKS IS MEASURED: a 100-mile fence around Bonsall reaches eight
// transmitters, which is why the fence is 100 and not the 25-mile service radius
// that would leave this control with one thing to select.
func TestTheBedSelectorWalksTheStationsFence(t *testing.T) {
	lp := bedPipelines(t)
	relays := lp.bedRelays()
	if len(relays) < 6 {
		t.Fatalf("the station's fence reaches a real choice; got %d", len(relays))
	}
	// NEAREST FIRST, and all of them inside the fence.
	for _, n := range relays {
		if mi := n.KM * 0.621371; mi > config.DefaultBedRadiusMi+0.001 {
			t.Errorf("%s is %.1f mi out and the fence is %v", n.Callsign, mi, config.DefaultBedRadiusMi)
		}
	}
	// AND A STATION WITH NO EPICENTRE HAS NOTHING TO CHOOSE BETWEEN, rather than
	// the whole country.
	bare := &livePipelines{relayTable: lp.relayTable}
	if got := bare.bedRelays(); len(got) != 0 {
		t.Errorf("no transmitter is no region; got %d relays", len(got))
	}
}

// THE ARROWS MOVE THE SELECTION, AND THEY WRAP.
//
// A SELECTOR THAT STOPPED AT THE ENDS would leave the operator pressing a key
// that does nothing and wondering which of the two reasons it was.
func TestSteppingTheBedWrapsBothWays(t *testing.T) {
	lp := bedPipelines(t)
	n := len(lp.bedRelays())
	if n < 2 {
		t.Fatalf("the fixture needs somewhere to step; got %d relays", n)
	}
	lp.stepBedRelay(-1)
	if lp.bedPick != n-1 {
		t.Errorf("stepping back from the first wraps to the last; got %d of %d", lp.bedPick, n)
	}
	lp.stepBedRelay(1)
	if lp.bedPick != 0 {
		t.Errorf("and forward from the last wraps to the first; got %d", lp.bedPick)
	}
	// A STATION WITH NO RELAYS IS A NO-OP, never a panic or a negative index.
	bare := &livePipelines{relayTable: lp.relayTable}
	bare.stepBedRelay(1)
	if bare.bedPick != 0 {
		t.Errorf("a fence with nothing in it selects nothing; got %d", bare.bedPick)
	}
}

// THE ROW READS THE WAY THE REFERENCE DRAWS IT.
//
//	[ ← ] KIG78 Coachella CA 162.400 MHz · 41mi from TOWER GPS  [ → ]
func TestTheRelayRowNamesTheTransmitter(t *testing.T) {
	lp := bedPipelines(t)
	got := relayLine(lp.bedRelays()[0])
	for _, want := range []string{"MHz", "mi from TOWER GPS"} {
		if !strings.Contains(got, want) {
			t.Errorf("the relay row is missing %q: %q", want, got)
		}
	}
	if strings.HasPrefix(got, " ") || strings.Contains(got, "  ") {
		t.Errorf("the row carries a stray gap: %q", got)
	}
}

// AND THE TWO PUBLISHERS AGREE ABOUT WHETHER THE BED IS CARRYING.
//
// The Director publishes it and the selector publishes it, and a selector that
// GUESSED would flicker the row between them. It is recorded from the Director,
// never decided here.
func TestTheSelectorReportsWhatTheDirectorSaidAboutCarrying(t *testing.T) {
	lp := bedPipelines(t)
	if lp.bedCarrying() {
		t.Error("a station that has been told nothing is not carrying")
	}
	lp.noteBedCarrying(true)
	if !lp.bedCarrying() {
		t.Error("what the Director said is what the selector reports")
	}
}
