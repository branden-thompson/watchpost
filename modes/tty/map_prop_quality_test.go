package tty

import (
	"errors"
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// TestOnlyTheLatestRefreshTickAsks is FR-4.10 (D-94): each result sets the
// next refresh tick, and only the latest set asks - an update U asked for
// starts no second chain of ticks beside the first.
func TestOnlyTheLatestRefreshTickAsks(t *testing.T) {
	s := &seam{answer: PropagationResult{Err: errors.New("offline")}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	d = runProp(t, d, cmd)
	first := propTickMsg{gen: d.mapPane.propGen, seq: d.mapPane.propTickSeq}
	d, cmd = keyCmd(t, d, "U")
	d = runProp(t, d, cmd)
	if s.count() != 2 {
		t.Fatalf("U asked %d in all; want a second ask", s.count())
	}
	m, cmd := d.Update(first)
	if d = runProp(t, m.(Dashboard), cmd); s.count() != 2 {
		t.Errorf("the tick set before U asked too: %d asks; one chain of ticks, not two", s.count())
	}
	m, cmd = d.Update(propTickMsg{gen: d.mapPane.propGen, seq: d.mapPane.propTickSeq})
	if runProp(t, m.(Dashboard), cmd); s.count() != 3 {
		t.Errorf("the latest tick asked %d in all; want a third ask", s.count())
	}
}

// TestAReleasedMapDropsTheOldSessionsTick is FR-4.10: a tick from a session
// before the map was released asks nothing in a session after it.
func TestAReleasedMapDropsTheOldSessionsTick(t *testing.T) {
	s := &seam{answer: PropagationResult{Err: errors.New("offline")}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	d = runProp(t, d, cmd)
	stale := propTickMsg{gen: d.mapPane.propGen, seq: d.mapPane.propTickSeq}
	d = d.toggleMap().releaseMap().toggleMap()
	d, cmd = keyCmd(t, feedAndSettle(t, d), "P")
	d = runProp(t, d, cmd)
	asked := s.count()
	m, cmd := d.Update(stale)
	if runProp(t, m.(Dashboard), cmd); s.count() != asked {
		t.Errorf("a tick from before the release asked again (%d asks, %d before)", s.count(), asked)
	}
}

// TestAReopenedMapDrawsNoPropagationField is D-154: every open is a weather
// mode, so a map reopened before it is released draws no field of the
// Propagation mode it was closed in.
func TestAReopenedMapDrawsNoPropagationField(t *testing.T) {
	d, _ := drawnDash(t)
	if len(d.mapPane.propGiven) != 1 || len(d.mapPane.nightGiven) != 1 {
		t.Fatalf("the fixture drew %d Propagation overlays and %d nights; want the field and the night", len(d.mapPane.propGiven), len(d.mapPane.nightGiven))
	}
	d = d.toggleMap().toggleMap()
	if d.mapMode() == modePropagation || len(d.mapPane.propGiven) != 0 || len(d.mapPane.nightGiven) != 0 {
		t.Errorf("reopened in mode %d with %d Propagation overlays and %d nights still drawn (Day / night is off)", d.mapMode(), len(d.mapPane.propGiven), len(d.mapPane.nightGiven))
	}
}

// TestThePropagationAgeMovesWithTheClock is D-166: the snapshot's age and
// its tone follow the clock with the window open and nothing else changing.
func TestThePropagationAgeMovesWithTheClock(t *testing.T) {
	at := time.Now().UTC().Truncate(time.Minute)
	clock := at.Add(20 * time.Second)
	s := &seam{answer: PropagationResult{Snapshot: propSnapAt(at, 18)}}
	var asked []string // radar configured, as the app's is: the block under the map is drawn only with its timeline
	d := openMap(t, Config{PropagationUpdate: s.update, PropagationAck: propAckVersion, MapRadar: radarFeed(t, "MRMS", &asked)}, 133, 44)
	d.now = func() time.Time { return clock }
	d, cmd := keyCmd(t, d, "P")
	d = runProp(t, d, cmd)
	d.mapPane.flash = "" // P's blink spent: the age alone needs the tick now, and the frame is memoised without it
	if !strings.Contains(stripANSITest(d.View().Content), "(data 30 min old)") {
		t.Fatal("the fixture's grid, valid 30 min before its snapshot, is not said 30 min old")
	}
	if !d.tickNeeded() {
		t.Error("no tick is armed while the age is shown: it would not move")
	}
	clock = at.Add(2*time.Minute + 10*time.Second)
	if view := stripANSITest(d.View().Content); !strings.Contains(view, "(data 32 min old)") {
		t.Errorf("two minutes on, the frame still says %q", between(view, "COMPUTED", "noise"))
	}
}

// between is the text of s from one mark to the next, for a short message.
func between(s, from, to string) string {
	i := strings.Index(s, from)
	if i < 0 {
		return ""
	}
	j := strings.Index(s[i:], to)
	if j < 0 {
		return s[i:]
	}
	return s[i : i+j]
}

// TestAWeatherTintFromPropagationAsksOnce is FR-4.2 and the Open-Meteo
// quota: a weather tint chosen in the Propagation mode asks the temperature
// once for the weather mode it returns to, never twice.
func TestAWeatherTintFromPropagationAsksOnce(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	d.cfg.PropagationUpdate = (&seam{answer: PropagationResult{Err: errors.New("offline")}}).update
	d.cfg.PropagationAck = propAckVersion
	d = pressMap(t, d, "P")
	before := len(asks)
	d = pressMap(t, menuOn(t, d, UVLayer), "space")
	if d.mapMode() != modeForecast || !d.tintChosen(UVLayer) {
		t.Fatalf("mode %d, UV chosen %v", d.mapMode(), d.tintChosen(UVLayer))
	}
	if n := len(asks) - before; n != 1 {
		t.Errorf("the tint asked the temperature %d times; want once", n)
	}
}

// TestTheNoteRowsAreCountedAsDrawn: the rows the map's size leaves for its
// notes are the notes drawn, in every mode, with a picture or without.
func TestTheNoteRowsAreCountedAsDrawn(t *testing.T) {
	d, _ := drawnDash(t)
	for _, mode := range []string{"propagation", "radar"} {
		if mode == "radar" {
			d = pressMap(t, d, "P")
		}
		for _, c := range []struct {
			ascii   bool
			instead bool
		}{{false, false}, {true, false}, {false, true}} {
			e := d
			e.cfg.ASCII = c.ascii
			if c.instead {
				e.mapDesc = mapDescInstead
			}
			for _, w := range []int{40, e.mapTextW()} {
				if got, want := e.noteRowCount(w), len(e.noteLines(w)); got != want {
					t.Errorf("%s, ASCII %v, description instead %v, width %d: %d note rows counted, %d drawn", mode, c.ascii, c.instead, w, got, want)
				}
			}
		}
	}
}

// TestTheSourcesBadgesSayWhatFedTheSnapshot is D-168 (A-42): before any
// snapshot every source's badge is dimmed; with one, PYIRI and IGRF-14 are
// lit, and SWPC is lit when foF2 is GloTEC's and dimmed when it is the
// climatology's.
func TestTheSourcesBadgesSayWhatFedTheSnapshot(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	climatology := propSnapAt(time.Now().UTC(), 18)
	climatology.Background.FoF2 = ionomaps.Climatology
	for _, c := range []struct {
		name string
		snap *ionomaps.Snapshot
		lit  map[string]bool
	}{
		{"no snapshot", nil, map[string]bool{"SWPC": false, "PYIRI": false, "IGRF-14": false}},
		{"foF2 from GloTEC", ptr(propSnapAt(time.Now().UTC(), 18)), map[string]bool{"SWPC": true, "PYIRI": true, "IGRF-14": true}},
		{"foF2 from the climatology", &climatology, map[string]bool{"SWPC": false, "PYIRI": true, "IGRF-14": true}},
	} {
		d := propDash(t, &seam{answer: PropagationResult{Err: errors.New("offline")}}, true)
		d.mapPane.prop, d.mapPane.propSnap = true, c.snap
		head := d.propBlock(d.scrubW())[0]
		for badge, lit := range c.lit {
			want := chipDimmed(badge)
			if lit {
				want = chipFace(badge)
			}
			if !strings.Contains(head, want) {
				t.Errorf("%s: %s is not %s: %q", c.name, badge, map[bool]string{true: "lit", false: "dimmed"}[lit], head)
			}
		}
	}
}

func ptr[T any](v T) *T { return &v }
