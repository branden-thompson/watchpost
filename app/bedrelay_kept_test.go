package app

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/platform/config"
)

// THE BED RELAY PICK IS KEPT BY ITS CALLSIGN (D-214, D-215): a step writes the
// chosen relay's callsign to the file, and the next launch selects it - once
// the relays near the station are known, and only while it is among them -
// without playing it. A list that moves under the selection keeps the relay,
// not the position.
func TestTheBedRelayPickIsKeptByItsCallsign(t *testing.T) {
	withConfigFile(t)
	lp := bedPipelines(t)
	relays := lp.bedRelays()
	if len(relays) < 3 {
		t.Fatalf("the fixture needs three relays; got %d", len(relays))
	}
	lp.deck = &radioDeck{}
	if cmd := lp.stepBedRelay(2); cmd != nil {
		cmd()
	}
	chosen := relays[2]
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Broadcaster.BedRelay != chosen.Callsign {
		t.Fatalf("the file keeps %q; the operator chose %q", cfg.Broadcaster.BedRelay, chosen.Callsign)
	}
	// THE NEXT LAUNCH: the pipelines know the kept callsign, and the list arrives.
	next := bedPipelines(t)
	next.deck = &radioDeck{}
	next.keepBedRelay(cfg)
	next.setBedStations(relays)
	if got := next.selectedRelay(); got != relayLine(chosen) {
		t.Errorf("the next launch selects %q; the file keeps %q", got, chosen.Callsign)
	}
	if next.deck.mode != "" || len(next.deck.mountURLs) != 0 {
		t.Error("the restore played the relay: it must only select it")
	}
	// AND THE LIST MOVES - reordered, the chosen relay still in it.
	moved := append([]stream.Station{chosen}, relays[:2]...)
	next.setBedStations(moved)
	if got := next.selectedRelay(); got != relayLine(chosen) {
		t.Errorf("after the list moved the row says %q; it is still %q", got, chosen.Callsign)
	}
	// AND THE RELAY LEAVES THE LIST: nothing is selected while it is not near.
	next.setBedStations(relays[:2])
	if got := next.selectedRelay(); got != "" {
		t.Errorf("a relay no longer near the station is still selected: %q", got)
	}
}

// THE BED PLAYS ON AIR AND ON A CUT, AND STOPS ON STANDBY (D-215): going on
// air, or cutting the programme to the bed, starts the selected relay; standby
// stops it - dead air is no relay playing.
func TestTheBedPlaysOnAirAndStopsOnStandby(t *testing.T) {
	plays, stops := 0, 0
	m := &mastercontrol{}
	m.playBed, m.stopBed = func() { plays++ }, func() { stops++ }
	m.GoOnAir()
	m.CutBed(true)
	m.CutBed(false)
	m.GoToStandby()
	if plays != 2 || stops != 1 {
		t.Errorf("on air, a cut to the bed and a cut back, then standby: %d plays, %d stops; want 2 and 1", plays, stops)
	}
}

// THE BED NEVER STARTS OVER A CARD BEING READ (D-216): the play key and going
// on air leave a read alone, a relay already playing is not restarted, and
// another relay playing is replaced by the chosen one.
func TestTheBedNeverStartsOverARead(t *testing.T) {
	lp := bedPipelines(t)
	relays := lp.bedRelays()
	lp.deck = &radioDeck{mode: "read"}
	if cmd := lp.stepBedRelay(1); cmd != nil {
		cmd()
	}
	if cmd := lp.toggleBedRelay(); cmd != nil {
		cmd()
	}
	lp.playBed()
	if lp.deck.mode != "read" || len(lp.deck.mountURLs) != 0 {
		t.Errorf("the deck is %q on %d mounts; the card being read must not be cut", lp.deck.mode, len(lp.deck.mountURLs))
	}

	chosen := relays[1]
	lead := chosen.Mounts[0].URL
	lp.deck = &radioDeck{mode: "live", mountURLs: []string{lead, "sentinel"},
		mountOwner: map[string]stream.Station{lead: chosen}}
	lp.playBed()
	if len(lp.deck.mountURLs) != 2 || lp.deck.mountURLs[1] != "sentinel" {
		t.Error("going on air restarted a relay already playing")
	}
	other := relays[0]
	lp.deck = &radioDeck{mode: "live", mountURLs: []string{other.Mounts[0].URL},
		mountOwner: map[string]stream.Station{other.Mounts[0].URL: other}}
	if cmd := lp.toggleBedRelay(); cmd != nil {
		cmd()
	}
	if lp.deck.mode != "live" || lp.deck.mountOwner[lp.deck.mountURLs[0]].Callsign != chosen.Callsign {
		t.Error("with another relay playing, the play key stopped it instead of playing the chosen one")
	}
	lp.stopBed()
	if lp.deck.mode != "" {
		t.Errorf("standby left the deck %q; want the relay stopped", lp.deck.mode)
	}
}
