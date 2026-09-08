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
	"fmt"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/platform/lineup"
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
	lib := script.New("")
	for _, burst := range []bool{true, false} {
		got := breakingLine(lib, fabricatedTornado(), burst, render.Clock12, execNow)
		if !strings.Contains(got, "A test Tornado Warning has been issued") {
			t.Errorf("burst=%v: the spoken line for a fabricated alert is %q, and nothing in it "+
				"says the alert is not real", burst, got)
		}
		if real := breakingLine(lib, tornado(), burst, render.Clock12, execNow); strings.Contains(real, "A test") {
			t.Errorf("burst=%v: a REAL alert is being read as a test event: %q", burst, real)
		}
	}
}

// AND THE MARKING IS NOT ONLY IN THE SCRIPT TEMPLATE, so no edit to a script
// file can remove it.  The read scripts are user-editable by design; a safety
// marking a listener can delete by emptying a template is not a marking.
func TestTheTestEventMarkingSurvivesABrokenScriptLibrary(t *testing.T) {
	got := breakingLine(brokenLibrary(t), fabricatedTornado(), true, render.Clock12, execNow)
	if !strings.Contains(got, "A test Tornado Warning has been issued") {
		t.Errorf("with the script gone the line reads %q: the marking lives only in the template, "+
			"so editing a text file removes the only thing saying the alert is fabricated", got)
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
	lib := script.New("")
	row := tornadoRow()
	row.Test = true
	got := eventScript(lib, row)
	if !strings.HasPrefix(got, testHead(lib)) {
		t.Errorf("the [w] report for a fabricated event reads %q", got)
	}
	if real := eventScript(lib, tornadoRow()); strings.Contains(real, "This is only a test") {
		t.Errorf("a REAL event's report is being read as a test: %q", real)
	}
}

// AND THE MARK SURVIVES THE TRIP FROM THE FEED TO THE WINDOW'S ROW.
func TestAFabricatedEventReachesTheSevereWindowMarked(t *testing.T) {
	got := toSevereRow(severe.Row{Key: "k", Product: "Tornado Warning", Test: true})
	if !got.Test {
		t.Error("the severe window's row lost the mark on the way from the feed")
	}
	// AND THE DETECTION COLUMN SAYS HOW IT WAS ESTABLISHED (HUM LEAD
	// 2026-09-07). That column is the row's own account of where the event came
	// from, and "injected" is the true answer for one the ctrl+d window made.
	if got.Detection != "injected" {
		t.Errorf("the fabricated row's detection reads %q, want \"injected\"", got.Detection)
	}
	if toSevereRow(severe.Row{Key: "k", Product: "Tornado Warning"}).Test {
		t.Error("a real row arrived at the window marked as a test")
	}
}

// A CARD OF FABRICATED ALERTS READS THE TEST SCRIPT (FR-4.5, HUM LEAD
// 2026-09-07).
//
// ALERT-AGNOSTIC, AND THAT IS THE POINT: the same words for every category, so
// exercising the machinery never waits on someone writing suitable content for
// each hazard. The tone still comes from the alert type the operator chose, so
// what is being tested — the tone, the takeover, the band, the window, the
// expiry — is the machinery and not the copy.
func TestAFabricatedCardReadsTheTestScript(t *testing.T) {
	lib := script.New("")
	e := fabricatedTornado()
	sc := composeTakeover(lib, []globalfeed.Event{e}, false, 0, render.Clock12, execNow)

	whole := partsText(sc)
	for _, want := range []string{
		"This is a test of the Watchpost alert events system",
		"A test Tornado Warning has been issued",
		"had this been a real Tornado Warning",
		"This concludes the test of the Watchpost alert events system",
	} {
		if !strings.Contains(whole, want) {
			t.Errorf("the test read does not say %q:\n%s", want, whole)
		}
	}
	// THE TONE IS THE ALERT'S OWN. The script is agnostic; the sound is not.
	if want := toneClassOfEvent(e).Key(); sc.Tone != want {
		t.Errorf("the test card sounds %q, want the injected type's own tone %q", sc.Tone, want)
	}
	// AND IT IS A HEAD, LINES AND A TAIL — the shape a real burst has, because
	// the shape is part of what is being tested.
	if sc.Parts[0].Kind != lineup.PartHead || sc.Parts[len(sc.Parts)-1].Kind != lineup.PartTail {
		t.Errorf("the test card is not head-lines-tail: %v", kindsOf(sc))
	}
}

// ONE ALERT, ONE TITLE; ONE EXPLANATION FOR THE CARD. Six fabricated alerts
// each carrying the sixty-word explanation is a seven-minute test read, and
// saying it six times is not more informative than saying it once.
func TestAFabricatedBurstSaysWhatItIsOnce(t *testing.T) {
	lib := script.New("")
	evs := []globalfeed.Event{fabricatedTornado(), fabricatedTornado(), fabricatedTornado()}
	for i := range evs {
		evs[i].ID = fmt.Sprintf("inj-%d", i)
	}
	sc := composeTakeover(lib, evs, true, 0, render.Clock12, execNow)

	if got := strings.Count(partsText(sc), "A test Tornado Warning has been issued"); got != 3 {
		t.Errorf("three fabricated alerts produced %d titles: each one is an alert and gets its own line", got)
	}
	if got := strings.Count(partsText(sc), "had this been a real"); got != 1 {
		t.Errorf("the explanation is said %d times; once is what it is worth", got)
	}
}

// AND A CARD THAT IS NOT ALL FABRICATED IS NOT A TEST READ. A real hazard in
// the same burst means the card is about real hazards: it keeps the ordinary
// head and tail, and the fabricated lines say what they are.
func TestAMixedCardIsReadAsTheRealThingWithTheTestLinesMarked(t *testing.T) {
	lib := script.New("")
	sc := composeTakeover(lib, []globalfeed.Event{tornado(), fabricatedTornado()}, true, 0, render.Clock12, execNow)

	whole := partsText(sc)
	if strings.Contains(whole, "This is a test of the Watchpost alert events system") {
		t.Errorf("a burst carrying a REAL tornado warning was read as a test of the system:\n%s", whole)
	}
	if !strings.Contains(whole, "A test Tornado Warning has been issued") {
		t.Errorf("the fabricated line in a real burst does not say it is a test:\n%s", whole)
	}
}

func partsText(sc lineup.Script) string {
	var b strings.Builder
	for _, p := range sc.Parts {
		b.WriteString(p.Text + "\n")
	}
	return b.String()
}

func kindsOf(sc lineup.Script) []lineup.PartKind {
	out := make([]lineup.PartKind, 0, len(sc.Parts))
	for _, p := range sc.Parts {
		out = append(out, p.Kind)
	}
	return out
}
