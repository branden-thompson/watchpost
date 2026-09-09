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
}

// snap is what a property needs to know about the schedule at one instant.
type snap struct {
	ids         map[string]bool // every card the schedule holds
	onAir       map[string]bool // every card in the OnAir state, so two can be SEEN
	byTrack     map[string]Track
	railWaiting bool // the rail holds a card still on its way to the air
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
	next, fx := m.d.Step(ev)
	m.d = next
	m.pending = append(m.pending, fx...)
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
		if after.byTrack[b.ID] == MainTrack && after.railUnprepared {
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
		// 4. THE RAIL DRAINS FIRST. Asserted at the MOMENT the programme takes
		// the air, against the schedule AS IT STANDS AFTER the step — because
		// the step that airs the programme is very often the same step that
		// emptied the rail, and `before` calls that a violation when it is the
		// rule working. The `after` reading is also the strict one: an
		// unready rail card blocks the air entirely (airOnce takes Next() or
		// nothing), so a hazard queued in this same step cannot be outrun.
		if after.byTrack[id] == MainTrack && after.railWaiting {
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
	var totalSteps, totalAdmits, totalAirs, reportAirs, railAirs, repeats int

	for run := range runs {
		rng := rand.New(rand.NewSource(int64(run)))
		m := &merged{
			t: t, rng: rng,
			d:      New(Settings{Max: 5, Watchlist: locations, Dwell: 2 * time.Minute}, base),
			admits: map[string]int{}, airs: map[string]int{},
		}
		now := base
		m.step(Powered{To: Running})

		for range 60 {
			// SERVICE OR ADVANCE, at random: whether the station gets to
			// finish what it started before the next thing happens IS the
			// timing dimension under test.
			if len(m.pending) > 0 && rng.Intn(2) == 0 {
				m.service()
				continue
			}
			now = now.Add(time.Duration(rng.Intn(90)) * time.Second)
			switch rng.Intn(7) {
			case 0, 1:
				ref := locations[rng.Intn(len(locations))]
				m.step(NeedsRead{Ref: ref, Headline: "REPORT FOR " + ref})
			case 2, 3:
				m.step(Arrived{Arrivals: randomArrivals(rng, now), Fence: Fence{}})
			case 4:
				m.step(Tick{Now: now})
			case 5:
				ref := locations[rng.Intn(len(locations))]
				m.step(Tuned{Ref: ref, Live: rng.Intn(2) == 0})
			case 6:
				// THE PROGRAMME STOPS AND STARTS UNDER LOAD. Stopping while a
				// hazard is on the rail is the case DR-3's asymmetry exists
				// for, and it must not be the way a report is read twice.
				if rng.Intn(4) == 0 {
					m.step(Powered{To: Stopped})
					m.step(Powered{To: Running})
					continue
				}
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
			if m.airs[id] > 1 {
				repeats++ // a location legitimately read again after its card left
			}
		}
		totalSteps += m.steps
	}

	// SILENCE IS A DISTINCT VERDICT (INST-2). Every one of these ran green
	// against a station that never read anything, so the run's REACH is
	// asserted, not assumed — and each figure names the property it feeds.
	t.Logf("%d runs, %d steps: %d admissions, %d readings (%d reports, %d hazards), %d locations read more than once",
		runs, totalSteps, totalAdmits, totalAirs, reportAirs, railAirs, repeats)
	if reportAirs == 0 {
		t.Error("no MAIN-TRACK card ever took the air: property 4 (the rail drains first) was never put to the test")
	}
	if railAirs == 0 {
		t.Error("no HAZARD ever took the air: the rail was never exercised against the merged producer")
	}
	if repeats == 0 {
		t.Error("no location was ever read a second time: property 3 (a report is never read twice) never " +
			"had to distinguish a legitimate second turn from a double read, which is the whole difficulty")
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
