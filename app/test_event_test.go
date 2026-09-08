package app

// test_event_test.go — FR-4.4: a fabricated alert is marked wherever it is
// perceived.
//
// A SCREENSHOT OF A FABRICATED TORNADO WARNING IS INDISTINGUISHABLE FROM A REAL
// ONE. That sentence is why the injector is compiled out of a release build,
// and it is just as true of a photograph of the marquee and of a recording of
// the read. The capability is absent from a release binary; the MARKING is not,
// because the surfaces that have to carry it — the band and the spoken line —
// ship in every build.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/platform/render"
)

func fabricatedTornado() globalfeed.Event {
	e := tornado()
	e.Fabricated = true
	return e
}

// IT SAYS SO ON EVERY LINE, not once at the top (FR-4.4). A listener who walks
// in halfway through a six-alert burst has heard no header, and the words are
// the only thing they have.
func TestEverySpokenLineForATestEventSaysItIsOne(t *testing.T) {
	for _, burst := range []bool{true, false} {
		got := breakingLine(nil, fabricatedTornado(), burst, render.Clock12, execNow)
		if !strings.Contains(got, testEventSpoken) {
			t.Errorf("burst=%v: the spoken line for a fabricated alert is %q, and nothing in it "+
				"says the alert is not real", burst, got)
		}
		if real := breakingLine(nil, tornado(), burst, render.Clock12, execNow); strings.Contains(real, testEventSpoken) {
			t.Errorf("burst=%v: a REAL alert is being read as a test event: %q", burst, real)
		}
	}
}

// AND THE MARKING IS NOT IN THE SCRIPT TEMPLATE, so no edit to a script file
// can remove it.  The read scripts are user-editable by design; a safety
// marking a listener can delete by overriding a template is not a marking.
func TestTheTestEventMarkingSurvivesABrokenScriptLibrary(t *testing.T) {
	// A library that renders NOTHING — the worst case a script edit can produce.
	got := breakingLine(brokenLibrary(t), fabricatedTornado(), true, render.Clock12, execNow)
	if !strings.Contains(got, testEventSpoken) {
		t.Errorf("with the script gone the line reads %q: the marking lives in the template, so "+
			"editing a text file removes the only thing saying the alert is fabricated", got)
	}
}

// AND IT TRAVELS TO THE BURST AS A TEST EVENT, so the selection can keep it
// from taking a real hazard's place (the rule lives in platform/lineup).
func TestAFabricatedEventArrivesAtTheBurstMarked(t *testing.T) {
	evs := []globalfeed.Event{fabricatedTornado(), tornado()}
	got := arrivalsOf(evs)
	if len(got) != 2 {
		t.Fatalf("two events in, %d arrivals out", len(got))
	}
	if !got[0].Test {
		t.Error("the fabricated event reached the burst as a real one")
	}
	if got[1].Test {
		t.Error("the real event reached the burst as a test one")
	}
}

// AND THE MARQUEE CARRIES IT (FR-4.4), as LANE CHROME rather than tape text:
// the tape is one scrolling line, so a marker inside it is off-window most of
// the time, and an 18-cell prefix per item at the 80-column floor makes the
// marker the majority of the tape.
func TestTheMarqueeCarriesTheTestEventMark(t *testing.T) {
	if !breakingItem(fabricatedTornado()).Test {
		t.Error("the fabricated event reached the band as a real one")
	}
	if breakingItem(tornado()).Test {
		t.Error("a REAL alert is on the band marked as a test event")
	}
}

// AND THE [w] REPORT SAYS IT FIRST (FR-4.4). The window's read is one script
// rather than a line per alert, so the marking leads it: a listener who asked
// for the full report on a fabricated event hears what it is before they hear
// what it says.
func TestTheEventReportSaysWhenItsSubjectIsFabricated(t *testing.T) {
	row := tornadoRow()
	row.Test = true
	got := eventScript(nil, row)
	if !strings.HasPrefix(got, testEventSpoken) {
		t.Errorf("the [w] report for a fabricated event reads %q", got)
	}
	if real := eventScript(nil, tornadoRow()); strings.Contains(real, testEventSpoken) {
		t.Errorf("a REAL event's report is being read as a test: %q", real)
	}
}

// AND THE MARK SURVIVES THE TRIP FROM THE FEED TO THE WINDOW'S ROW.
func TestAFabricatedEventReachesTheSevereWindowMarked(t *testing.T) {
	if !toSevereRow(severe.Row{Key: "k", Product: "Tornado Warning", Test: true}).Test {
		t.Error("the severe window's row lost the mark on the way from the feed")
	}
	if toSevereRow(severe.Row{Key: "k", Product: "Tornado Warning"}).Test {
		t.Error("a real row arrived at the window marked as a test")
	}
}
