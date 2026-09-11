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
// THE DEFECT (HUM LEAD, UAT 2026-09-10): "audio in Broadcaster is still pulling
// audio from Observer", and its sibling — `ctrl+o` refused after a round trip to
// Observer and back (D-69). ONE `power` field served two programmes, so a tune
// in Observer put the CONSOLE on the air.
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
// They advanced on ONE gate — `advances(MainTrack)` — so a running station could
// produce both at once: the line-up the operator scheduled, and the watchlist
// they had been listening to underneath it.
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
	lp.takeTheAir(tty.SurfaceBroadcaster)

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
	lp.takeTheAir(tty.SurfaceBroadcaster)

	var told []lineup.Event
	deck.emit = func(ev lineup.Event) { told = append(told, ev) }
	lp.takeTheAir(tty.SurfaceObserver)

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
	// THE PLANT THAT FOUND THIS (`y8`) deleted `HandAir(AirMonitor)` and
	// SURVIVED: the deck reads the Router's own owner, so it would go on playing
	// while the Director still believed the console held the air — and the
	// watchlist would never advance again. Two carriers of one fact, and only
	// one of them was asserted.
	nar := testDirector(nil, func(tea.Msg) {})
	var declared []lineup.Event
	nar.mc.mu.Lock()
	nar.mc.carry = func(ev lineup.Event) { declared = append(declared, ev) }
	nar.mc.mu.Unlock()
	back := &livePipelines{director: nar, deck: deck, ticker: &tickerDeck{rescope: make(chan struct{}, 1)}}
	back.owner.set(tty.SurfaceBroadcaster)
	back.takeTheAir(tty.SurfaceObserver)
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
// THIS IS THE DEFECT ITSELF — "audio in Broadcaster is still pulling audio from
// Observer" — and the first version of this file did not assert it. It checked
// `monitorHasTheAir()`, the PREDICATE, and the plant that deleted the guard in
// `needsRead` SURVIVED. Exactly the shape `x5` had one ruling earlier: the
// assertion stopped at the question and never reached the answer.
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
// THE PLANT THAT FOUND THIS (`y6`) deleted `mc.StopMonitor()` and SURVIVED,
// because `deck.Stop()` reports `Monitored{false}` too — so with a deck present
// the two are indistinguishable. They are not the same thing: with no deck there
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
	lp.takeTheAir(tty.SurfaceBroadcaster)

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
// THE HUM LEAD ASKED WHETHER THE RAIL FILTERS AND EXPANDS WITH THE MODE. New
// arrivals did; everything already on the rail did not, because the Director
// learned a fence only on the next ARRIVAL. It learns it on the air now — and
// from `tickerDeck.fence()`, which is the same translation every arrival already
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
	lp.takeTheAir(tty.SurfaceBroadcaster)

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
	// AND A DECK WITH NO RAIL IS "ALL", never a panic — the older tests and the
	// pathless build, the same rule `fence()` itself states.
	bare := &mastercontrol{}
	if bare.railFence().InForce() {
		t.Error("an effector with no fence to ask is unfenced, not fenced at zero")
	}
}
