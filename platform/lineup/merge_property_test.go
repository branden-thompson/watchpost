package lineup

// THE MERGE'S PROPERTIES, ACROSS RANDOMISED TIMING (0.16.0 P3).
//
// A SCENARIO UAT TARGETS PRECEDENCE AND DUPLICATION, NOT TIMING. That was the
// safety lens's objection to P3's controls and it is right: a fixed scenario
// cannot reach the case where a cycle ends at the same tick an alert arrives,
// which is exactly where a merged producer breaks. So the rules are asserted
// over randomly interleaved arrivals, needs, ticks, completions and power
// changes, with a driver that also services effects at random moments — because
// "the build came home late" is itself a timing case.
//
// FOUR PROPERTIES, CHECKED AFTER EVERY STEP:
//
//  1. At most one card holds the air.  Two would be two voices.
//  2. No two cards in the schedule share an identity.
//  3. No card takes the air more times than it was admitted (FR-2.5): a report
//     is never read twice.
//  4. The alert rail drains first (DR-3): a main-track card may only take the
//     air at a moment when no rail card was waiting for it.
//
// COUNTED, NOT OBSERVED AT THE END. The schedule is a value, so every step's
// before and after are both in hand, and a transition can be asserted rather
// than inferred from where things ended up.

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
)

// randomArrivals is a burst of one to four hazards, of randomly chosen
// categories and severities, each already a little old.
//
// THE IDS REPEAT ACROSS BURSTS ON PURPOSE. Two bursts carrying the same alert
// is the case the Director's identity check and the producer's read store both
// exist for, and a generator that made every id unique would never produce it.
func randomArrivals(rng *rand.Rand, now time.Time) []Arrival {
	cats := category.ReadOrder()
	n := 1 + rng.Intn(4)
	out := make([]Arrival, 0, n)
	seen := map[string]bool{}
	for range n {
		id := fmt.Sprintf("alert-%02d", rng.Intn(6))
		if seen[id] {
			continue // no two arrivals in ONE burst may share an identity
		}
		seen[id] = true
		out = append(out, Arrival{
			ID: id, Category: cats[rng.Intn(len(cats))], Headline: id + " headline",
			Subject: "Bonsall, CA", Severity: rng.Intn(100),
			At:  now.Add(-time.Duration(rng.Intn(48)) * time.Hour),
			Lat: 33.2887, Lon: -117.2253, HasPoint: true,
		})
	}
	return out
}

// merged is one randomised run of the merged station.
type merged struct {
	t       *testing.T
	rng     *rand.Rand
	d       Director
	pending []Effect

	admits map[string]int // card id -> times it entered the schedule
	airs   map[string]int // card id -> times it took the air
	steps  int

	// whileStopped counts events stepped against a stopped Director. It is a
	// REACH figure, not a property: the driver used to stop and start in one
	// atomic pair, so this was structurally zero and the merge's whole power
	// asymmetry went untested (red team finding 3).
	whileStopped int
}

// snap is what a property needs to know about the schedule at one instant.
type snap struct {
	ids         map[string]bool  // every card the schedule holds
	onAir       map[string]bool  // every card in the OnAir state, so two can be SEEN
	byTrack     map[string]Track // a card the schedule no longer holds is ABSENT, never MainTrack
	railWaiting bool             // the rail holds a card still on its way to the air
	// railUnprepared is the rail holding a card whose words have not even been
	// asked for. A stricter reading than railWaiting, and the one property 5
	// needs: preparation is the expensive step.
	railUnprepared bool
}

func (m *merged) snapshot() snap {
	s := snap{ids: map[string]bool{}, onAir: map[string]bool{}, byTrack: map[string]Track{}}
	for t := range m.d.lineup.tracks { // bounded by the array (P10-02)
		for _, c := range m.d.lineup.tracks[t] {
			s.ids[c.ID] = true
			s.byTrack[c.ID] = Track(t)
			if c.State == OnAir {
				s.onAir[c.ID] = true
			}
			if Track(t) == AlertRail && (c.State == Admitted || c.State == Standby) {
				s.railWaiting = true
			}
			if Track(t) == AlertRail && c.State == Admitted {
				s.railUnprepared = true
			}
		}
	}
	return s
}

// step feeds one event and checks every property against the transition.
func (m *merged) step(ev Event) {
	before := m.snapshot()
	if m.d.Power() != Running {
		if _, ordinary := ev.(Powered); !ordinary {
			m.whileStopped++
		}
	}
	next, fx := m.d.Step(ev)
	m.d = next
	// ONLY WHAT THE DRIVER CAN ANSWER GOES INTO THE BACKLOG (red team blind
	// spot 7, measured). Every effect used to be queued and every service call
	// drew one at random, but only a BuildCard or a Speak comes home as an
	// event — and a Publish is emitted on EVERY settle, so the queue filled
	// with effects the driver could only discard. Measured: 29,798 service
	// calls, 4,680 of them actionable — 15.7% — and the ratio worsens through a
	// run, which makes the DELAYED BUILD this test exists for rarer the longer
	// it goes.
	//
	// The others are dropped rather than queued because that is what they are:
	// a publish is a send to a console, a cue and a release are the band's, a
	// tune is the deck's. None of them answers the Director back.
	for _, f := range fx { // bounded by the step's effects (P10-02)
		switch f.(type) {
		case BuildCard, Speak:
			m.pending = append(m.pending, f)
		}
	}
	m.steps++
	after := m.snapshot()

	// 1. AT MOST ONE CARD HOLDS THE AIR. Counted here rather than asked of
	// OnAir, which returns "nobody" when its own invariant trips — a violation
	// would otherwise read as an idle station.
	if len(after.onAir) > 1 {
		m.t.Fatalf("step %d (%T): two cards hold the air at once: %v — two voices", m.steps, ev, keys(after.onAir))
	}
	// 2. NO TWO CARDS SHARE AN IDENTITY. An ambiguous address is a card read
	// twice, and the schedule is what has to refuse it.
	m.checkUnique(ev)

	// 5. THE RAIL IS PREPARED FIRST, not merely aired first — FOUND BY A PLANT
	// THAT SURVIVED. Reversing the precedence in `Next` is caught by property
	// 4; reversing it in `toPrepare` was not, and it is the more dangerous of
	// the two. Preparation is the expensive step (a cold build is ~1 s of
	// network), and an unready rail card BLOCKS the air entirely — so composing
	// a report ahead of a waiting hazard does not merely reorder the reads, it
	// holds the whole station silent while the hazard queues behind a weather
	// report. Property 4 cannot see it: nothing takes the air at all.
	for _, f := range fx {
		b, ok := f.(BuildCard)
		if !ok {
			continue
		}
		// ASKED WITH ok, NOT BY VALUE. MainTrack is Track(0), so a card the
		// schedule no longer holds would read as a main-track card and produce
		// a failure nobody could explain (red team finding 11).
		tr, held := after.byTrack[b.ID]
		if held && tr == MainTrack && after.railUnprepared {
			m.t.Fatalf("step %d (%T): the programme card %q was sent to be composed while a hazard on the rail "+
				"had not been; the expensive step went to the wrong card and the air is blocked until it returns",
				m.steps, ev, b.ID)
		}
	}

	for id := range after.ids {
		if !before.ids[id] {
			m.admits[id]++
		}
	}
	for id := range after.onAir {
		if before.onAir[id] {
			continue // still reading; not a new taking of the air
		}
		m.airs[id]++
		// 3. FR-2.5 AT THE SCHEDULE LEVEL. A card may be read once per
		// admission — the rotation legitimately comes round to a location
		// again once its previous card has left — and never more.
		if m.airs[id] > m.admits[id] {
			m.t.Fatalf("step %d (%T): card %q took the air %d time(s) but was admitted %d — a report read twice",
				m.steps, ev, id, m.airs[id], m.admits[id])
		}
		// 6. A STOPPED PROGRAMME DOES NOT READ (PD-1, DR-3's asymmetry). The
		// rail advances while the programme is stopped; the main track must
		// not. Asked of the DIRECTOR's own power rather than of the driver's
		// bookkeeping, so a test that lost track of the station cannot make
		// this pass.
		if tr, held := after.byTrack[id]; held && tr == MainTrack && m.d.Power() != Running {
			m.t.Fatalf("step %d (%T): the programme card %q took the air while the station was %v",
				m.steps, ev, id, m.d.Power())
		}
		// 4. THE RAIL DRAINS FIRST. Asserted at the MOMENT the programme takes
		// the air, against the schedule AS IT STANDS AFTER the step — because
		// the step that airs the programme is very often the same step that
		// emptied the rail, and `before` calls that a violation when it is the
		// rule working. The `after` reading is also the strict one: an
		// unready rail card blocks the air entirely (airOnce takes Next() or
		// nothing), so a hazard queued in this same step cannot be outrun.
		tr, held := after.byTrack[id]
		if held && tr == MainTrack && after.railWaiting {
			m.t.Fatalf("step %d (%T): the programme card %q took the air while a hazard was waiting on the rail",
				m.steps, ev, id)
		}
	}
}

func (m *merged) checkUnique(ev Event) {
	seen := map[string]int{}
	for t := range m.d.lineup.tracks { // bounded by the array (P10-02)
		for _, c := range m.d.lineup.tracks[t] {
			seen[c.ID]++
			if seen[c.ID] > 1 {
				m.t.Fatalf("step %d (%T): the schedule holds %d cards with the identity %q", m.steps, ev, seen[c.ID], c.ID)
			}
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m { // bounded by the map (P10-02)
		out = append(out, k)
	}
	return out
}

// service performs one pending effect, chosen at random, and feeds its result
// back. THE DELAY IS THE POINT: a build that comes home three steps after it
// was asked for is the interleaving a fixed scenario cannot produce.
func (m *merged) service() {
	if len(m.pending) == 0 {
		return
	}
	i := m.rng.Intn(len(m.pending))
	f := m.pending[i]
	m.pending = append(m.pending[:i], m.pending[i+1:]...)
	switch v := f.(type) {
	case BuildCard:
		if m.rng.Intn(8) == 0 {
			// The composer failed — a report for a location the listener has
			// since removed, an alert the producer no longer holds.
			m.step(Failed{ID: v.ID, Reason: "composed nothing", Routed: true})
			return
		}
		m.step(Built{ID: v.ID, Script: Say("the words for " + v.ID)})
	case Speak:
		if m.rng.Intn(6) == 0 {
			m.step(Failed{ID: v.ID, Reason: "cut short", Routed: true})
			return
		}
		m.step(Finished{ID: v.ID})
	}
	// Every other effect is work outside the Director — a cue, a duck, a
	// publish, a tune — and comes home, if at all, as one of the events the
	// generator already produces.
}

func TestTheMergedStationHoldsItsPropertiesUnderRandomTiming(t *testing.T) {
	locations := []string{"oceanside", "bonsall", "vista"}
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	const runs = 300
	var totalSteps, totalAdmits, totalAirs, reportAirs, railAirs, repeats, stoppedEvents int

	for run := range runs {
		rng := rand.New(rand.NewSource(int64(run)))
		m := &merged{
			t: t, rng: rng,
			d:      New(Settings{Max: 5, Watchlist: locations, Dwell: 2 * time.Minute}, base),
			admits: map[string]int{}, airs: map[string]int{},
		}
		now := base
		m.step(Aired{To: AirProgramme})
		m.step(Powered{To: Running})
		stopped := false

		for range 200 {
			// SERVICE OR ADVANCE, at random: whether the station gets to
			// finish what it started before the next thing happens IS the
			// timing dimension under test.
			if len(m.pending) > 0 && rng.Intn(2) == 0 {
				m.service()
				continue
			}
			now = now.Add(time.Duration(rng.Intn(90)) * time.Second)
			// A STOP DOES NOT LAST FOR EVER. Left to case 7 alone the station
			// was down 44% of the time and the main track barely aired, so the
			// resume gets its own faster path — the stop is still a state with
			// duration, just a shorter one.
			if stopped && rng.Intn(3) == 0 {
				m.step(Aired{To: AirProgramme})
				m.step(Powered{To: Running})
				stopped = false
				continue
			}
			switch rng.Intn(8) {
			case 0, 1, 2:
				ref := locations[rng.Intn(len(locations))]
				m.step(NeedsRead{Ref: ref, Headline: "REPORT FOR " + ref})
			case 3:
				m.step(Arrived{Arrivals: randomArrivals(rng, now), Fence: Fence{}})
			case 4, 5:
				m.step(Tick{Now: now})
			case 6:
				ref := locations[rng.Intn(len(locations))]
				m.step(Tuned{Ref: ref, Live: rng.Intn(2) == 0})
			case 7:
				// A STOP IS A STATE WITH DURATION, NOT AN INSTANT (red team
				// finding 3). This used to step Stopped and Running as an
				// atomic pair, so across 300 runs and 2,630 needs, NOT ONE
				// ordinary event was ever stepped against a stopped Director —
				// and the whole asymmetry the merge turns on (DR-3: the rail
				// advances while the programme is stopped, the main track does
				// not) went unexercised. A neutralised PD-1 guard passed every
				// property.
				//
				// Now the station stays down until a later iteration flips it
				// back, so needs, arrivals, ticks and completions all land
				// while it is off.
				if stopped {
					m.step(Aired{To: AirProgramme})
					m.step(Powered{To: Running})
					stopped = false
					continue
				}
				m.step(Powered{To: Stopped})
				stopped = true
			default:
				m.step(Ended{})
			}
		}
		for id, n := range m.airs {
			totalAirs += n
			if isRead(id) {
				reportAirs += n
			} else {
				railAirs += n
			}
		}
		for id, n := range m.admits {
			totalAdmits += n
			// A LOCATION, NOT A HAZARD (red team finding 4). This counted every
			// card re-aired, and 35 of the 38 it reported were rail cards from
			// a repeated burst — so the gate below could have stayed green with
			// the rotation never once coming round, which is the exact case
			// property 3 exists to tell apart from a double read. The same
			// isRead the report counter uses, twelve lines up.
			if m.airs[id] > 1 && isRead(id) {
				repeats++
			}
		}
		totalSteps += m.steps
		stoppedEvents += m.whileStopped
	}

	// SILENCE IS A DISTINCT VERDICT (INST-2). Every one of these ran green
	// against a station that never read anything, so the run's REACH is
	// asserted, not assumed — and each figure names the property it feeds.
	t.Logf("%d runs, %d steps: %d admissions, %d readings (%d reports, %d hazards), %d locations read more than once, %d events on a stopped station",
		runs, totalSteps, totalAdmits, totalAirs, reportAirs, railAirs, repeats, stoppedEvents)
	if stoppedEvents == 0 {
		t.Error("not one event was ever stepped against a STOPPED station: property 6 and the producer's " +
			"own stopped-station gate were never exercised, whatever the other figures say")
	}
	if reportAirs == 0 {
		t.Error("no MAIN-TRACK card ever took the air: property 4 (the rail drains first) was never put to the test")
	}
	if railAirs == 0 {
		t.Error("no HAZARD ever took the air: the rail was never exercised against the merged producer")
	}
	// A FLOOR WITH ROOM IN IT. One occurrence would make this a presence check
	// that a seed change could take to zero while still reading as a reach
	// assertion; the case must be routine, not lucky.
	if repeats < 20 {
		t.Errorf("only %d LOCATION(s) were read a second time: property 3 (a report is never read twice) "+
			"barely had to distinguish a legitimate second turn from a double read, which is the whole "+
			"difficulty. Lengthen the runs or widen the rotation until this case is routine", repeats)
	}
}

// isRead says whether a card id names a rotation report rather than a hazard.
// DERIVED FROM THE ONE OWNER of that rule, so a change to how a rotation card
// is named cannot leave this counting the wrong thing (INST-1).
func isRead(id string) bool { return id == ReadID(trimReadPrefix(id)) }

func trimReadPrefix(id string) string {
	const p = "read:"
	if len(id) > len(p) && id[:len(p)] == p {
		return id[len(p):]
	}
	return id
}
