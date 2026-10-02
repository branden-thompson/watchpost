package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// LISTENING IS NOT BROADCASTING (D-74).
//
// THE PROPERTY: Broadcaster never plays audio pulled for Observer, and `ctrl+o`
// still works after a round trip to Observer and back (D-69). One `power` field
// serving both programmes would let a tune in Observer put the CONSOLE on the
// air.
func TestATuneStartsTheMonitorAndNotTheStation(t *testing.T) {
	d, _ := offlineDeck(t)
	dir := lineup.New(lineup.Settings{Max: 5}, time.Now())
	d.emit = func(ev lineup.Event) { dir, _ = dir.Step(ev) }

	d.tune(pinRef("A", 33.19, -117.37))
	d.engine.Halt()

	if !dir.MonitorRunning() {
		t.Error("a tune starts the operator's own listening")
	}
	if dir.Power() == lineup.Running {
		t.Error("and it does NOT put their station on the air")
	}
}

// THE TWO PROGRAMMES ARE MUTUALLY EXCLUSIVE, WHICH IS THE WHOLE POINT.
//
// ONE gate for both — `advances(MainTrack)` — would let a running station
// produce both at once: the line-up the operator scheduled, and the watchlist
// they listen to underneath it.
func TestOnlyOneProgrammeCanEverAdvance(t *testing.T) {
	for _, c := range []struct {
		air              lineup.Air
		monitor, station bool
		why              string
	}{
		{lineup.AirMonitor, true, false, "the operator is listening"},
		{lineup.AirProgramme, false, true, "the station is broadcasting"},
	} {
		d := lineup.New(lineup.Settings{Max: 5}, time.Now())
		d, _ = d.Step(lineup.Monitored{Running: true})
		d, _ = d.Step(lineup.Aired{To: c.air})
		d, _ = d.Step(lineup.Powered{To: lineup.Running})
		if got := d.MonitorAdvancesForTest(); got != c.monitor {
			t.Errorf("%s: the monitor advances = %v, want %v", c.why, got, c.monitor)
		}
		if got := d.StationAdvancesForTest(); got != c.station {
			t.Errorf("%s: the station advances = %v, want %v", c.why, got, c.station)
		}
	}
}

// MOVING TO THE CONSOLE TAKES THE AIR AND SILENCES THE MONITOR.
//
//	"If I'm in the Broadcaster UI, then the Observer Mode must be Silent."
//	 — HUM LEAD, 2026-09-10
//
// BOTH HALVES ARE NEEDED. The `Monitored` event stops the ROTATION; the deck's
// own stop ends what is ALREADY PLAYING. A rotation that will not advance still
// leaves the current read on the air.
func TestMovingToTheConsoleSilencesTheMonitor(t *testing.T) {
	deck, _ := offlineDeck(t)
	var told []lineup.Event
	deck.emit = func(ev lineup.Event) { told = append(told, ev) }

	lp := &livePipelines{deck: deck, ticker: &tickerDeck{rescope: make(chan struct{}, 1)}}
	// WIRED THE WAY PRODUCTION WIRES IT, so what is asserted is the seam the app
	// actually uses rather than a fixture's own answer.
	deck.air = func() bool { return lp.owner.get() != tty.SurfaceBroadcaster }
	// AND THE COMMAND IS RUN, which is what Bubble Tea does with it (D-79).
	// Silencing the monitor halts the player, so it CANNOT happen inline — a
	// test that never ran the command would assert the old, frozen shape.
	if cmd := lp.takeTheAir(tty.SurfaceBroadcaster); cmd != nil {
		cmd()
	}

	// THE DECK'S ROTATION IS ENDED, which is what `Stop` reports.
	stopped := false
	for _, ev := range told {
		if m, ok := ev.(lineup.Monitored); ok && !m.Running {
			stopped = true
		}
	}
	if !stopped {
		t.Errorf("taking the air to the console did not stop the monitor; told %v", told)
	}
	// AND THE DECK NO LONGER PLAYS A NEED, which is the half the operator hears.
	if deck.monitorHasTheAir() {
		t.Error("the deck still believes the monitor has the air")
	}
}

// GOING BACK DOES NOT RESUME IT.
//
//	"Stay silent; while air was taken — it was a result of the Operator action
//	 (ctrl+b -> ctrl+o) so I am okay with it 'restoring' state, like if Observer
//	 was first opened (with reasonable performance consideration in mind, we
//	 don't want to slam APIs by rapid user switching)."
//
// THE PERFORMANCE HALF IS THE REASON IT IS ASSERTED. A swap that resumed would
// re-resolve a relay and re-fetch its products every time the operator flipped
// between surfaces; staying silent is what makes a rapid ctrl+b / ctrl+o flip
// cost nothing but a re-scope.
func TestGoingBackToObserverDoesNotResumeTheMonitor(t *testing.T) {
	deck, _ := offlineDeck(t)
	lp := &livePipelines{deck: deck, ticker: &tickerDeck{rescope: make(chan struct{}, 1)}}
	deck.air = func() bool { return lp.owner.get() != tty.SurfaceBroadcaster }
	if cmd := lp.takeTheAir(tty.SurfaceBroadcaster); cmd != nil {
		cmd()
	}

	var told []lineup.Event
	deck.emit = func(ev lineup.Event) { told = append(told, ev) }
	if cmd := lp.takeTheAir(tty.SurfaceObserver); cmd != nil {
		cmd()
	}

	for _, ev := range told {
		if m, ok := ev.(lineup.Monitored); ok && m.Running {
			t.Errorf("returning to Observer started the monitor by itself; told %v", told)
		}
	}
	// THE AIR COMES BACK, so the operator's next press plays.
	if !deck.monitorHasTheAir() {
		t.Error("the monitor never got the air back; the operator's play would be inert")
	}
	// AND THE DIRECTOR IS TOLD, which is the half that keeps the ROTATION alive.
	//
	// THE DECK READS THE ROUTER'S OWN OWNER, so without `HandAir(AirMonitor)` it
	// goes on playing while the Director still believes the console holds the
	// air — and the watchlist never advances again. Two carriers of one fact,
	// so both are asserted.
	nar := testDirector(nil, func(tea.Msg) {})
	var declared []lineup.Event
	nar.mc.mu.Lock()
	nar.mc.carry = func(ev lineup.Event) { declared = append(declared, ev) }
	nar.mc.mu.Unlock()
	back := &livePipelines{director: nar, deck: deck, ticker: &tickerDeck{rescope: make(chan struct{}, 1)}}
	back.owner.set(tty.SurfaceBroadcaster)
	if cmd := back.takeTheAir(tty.SurfaceObserver); cmd != nil {
		cmd()
	}
	handed := false
	for _, ev := range declared {
		if a, ok := ev.(lineup.Aired); ok && a.To == lineup.AirMonitor {
			handed = true
		}
	}
	if !handed {
		t.Errorf("the Director was never told the monitor has the air again; declared %v", declared)
	}
}

// A TUNE WHILE THE CONSOLE HOLDS THE AIR REPORTS THE NEED AND STARTS NO AUDIO.
//
// THIS IS THE PROPERTY ITSELF — Broadcaster never pulls audio from Observer —
// so it asserts the ANSWER, not the question: checking `monitorHasTheAir()`,
// the PREDICATE, alone lets a deletion of the guard in `needsRead` pass.
//
// THE NEED IS STILL REPORTED, which is the other half. It is a FACT — nobody is
// carrying this location — and the Director is entitled to it whoever is on the
// air; what the air decides is who PERFORMS.
func TestATuneWhileTheConsoleHasTheAirIsSilentButStillReports(t *testing.T) {
	t.Setenv("WATCHPOST_MAINTRACK", "dark")
	deck, _ := offlineDeck(t)
	var told []lineup.Event
	deck.emit = func(ev lineup.Event) { told = append(told, ev) }
	deck.pref = tty.ModeSynth
	deck.air = func() bool { return false } // the console has it

	deck.tune(pinRef("A", 33.19, -117.37))
	deck.engine.Halt()

	// NO AUDIO. `startSynth` is the only thing that sets the deck's mode, so an
	// empty mode is the observable for "nothing began to play".
	deck.mu.Lock()
	mode := deck.mode
	deck.mu.Unlock()
	if mode != "" {
		t.Errorf("the deck started playing %q while the console held the air", mode)
	}
	// AND THE NEED REACHED THE DIRECTOR ANYWAY.
	reported := false
	for _, ev := range told {
		if _, ok := ev.(lineup.NeedsRead); ok {
			reported = true
		}
	}
	if !reported {
		t.Errorf("the need was never reported; the deck reports, the Director decides. told %v", told)
	}
}

// AND A STATION WITH NO AUDIO AT ALL STILL STOPS ITS ROTATION.
//
// WITH A DECK PRESENT, `deck.Stop()` reports `Monitored{false}` too, so it and
// `mc.StopMonitor()` are indistinguishable there, and only a deckless station
// tells them apart. They are not the same thing: with no deck there
// is nothing to stop, and only the declaration keeps the Director from going on
// rotating for a listener who is not there.
//
// It is a PRECONDITION, not a duplicate — the separating question being "could a
// different caller make this false", and a pathless build is that caller.
func TestTakingTheAirStopsTheRotationEvenWithNoAudio(t *testing.T) {
	var told []lineup.Event
	nar := testDirector(nil, func(tea.Msg) {})
	nar.mc.mu.Lock()
	nar.mc.carry = func(ev lineup.Event) { told = append(told, ev) }
	nar.mc.mu.Unlock()

	lp := &livePipelines{director: nar, ticker: &tickerDeck{rescope: make(chan struct{}, 1)}} // no deck
	if cmd := lp.takeTheAir(tty.SurfaceBroadcaster); cmd != nil {
		cmd()
	}

	stopped := false
	for _, ev := range told {
		if m, ok := ev.(lineup.Monitored); ok && !m.Running {
			stopped = true
		}
	}
	if !stopped {
		t.Errorf("a station with no audio still stops its rotation; told %v", told)
	}
}

// THE FENCE TRAVELS WITH THE AIR, AND IT IS THE ONE THE DECK COMPUTES (D-75).
//
// THE RAIL FILTERS AND EXPANDS WITH THE MODE — every card already on it, not
// only new arrivals — because the Director learns the fence on the air, not on
// the next ARRIVAL. It learns it from `tickerDeck.fence()`, which is the same
// translation every arrival already
// carries, so the fence that re-tests a card and the fence that admitted it
// cannot be two different readings of one setting.
func TestTheAirCarriesTheFenceTheRailIsScopedTo(t *testing.T) {
	var told []lineup.Event
	nar := testDirector(nil, func(tea.Msg) {})
	nar.mc.mu.Lock()
	nar.mc.carry = func(ev lineup.Event) { told = append(told, ev) }
	nar.mc.fence = func() lineup.Fence {
		return lineup.Fence{RadiusMi: 25, Lat: 33.2881, Lon: -117.2256, HasOrigin: true}
	}
	nar.mc.mu.Unlock()

	lp := &livePipelines{director: nar, ticker: &tickerDeck{rescope: make(chan struct{}, 1)}}
	if cmd := lp.takeTheAir(tty.SurfaceBroadcaster); cmd != nil {
		cmd()
	}

	var handed *lineup.Aired
	for i := range told {
		if a, ok := told[i].(lineup.Aired); ok {
			handed = &a
		}
	}
	if handed == nil {
		t.Fatalf("the air was never handed over; told %v", told)
	}
	if !handed.Fence.InForce() || handed.Fence.RadiusMi != 25 {
		t.Errorf("the air carried %+v, want the station's own 25-mile fence", handed.Fence)
	}
	// AND A DECK WITH NO RAIL IS "ALL", never a panic — tests that build no rail
	// and the pathless build, the same rule `fence()` itself states.
	bare := &mastercontrol{}
	if bare.railFence().InForce() {
		t.Error("an effector with no fence to ask is unfenced, not fenced at zero")
	}
}
