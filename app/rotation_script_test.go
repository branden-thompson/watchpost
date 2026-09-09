package app

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// P3(a): a location report's SEGMENTS become a card's SCRIPT.
//
// The two paths to speech compose into different shapes — the rotation makes
// []synth.Segment and plays it on the engine; a card carries lineup.Script and
// is read through the arbiter. S0 named this adapter as the join. PartLine's
// own comment already says it is "the whole of a location report", so the
// model anticipated the card; what was missing was the translation.

func TestSegmentsBecomeAScriptOfLines(t *testing.T) {
	segs := []synth.Segment{
		{Key: "obs", Text: "Currently sixty-one degrees.", Role: cast.Weather},
		{Key: "tail", Text: "This is Watchpost Weather Radio.", Role: cast.Station, SelfIntro: true},
	}
	got := scriptFromSegments(segs)

	if len(got.Parts) != len(segs) {
		t.Fatalf("every segment is a part: got %d parts for %d segments", len(got.Parts), len(segs))
	}
	for i, p := range got.Parts {
		if p.Kind != lineup.PartLine {
			t.Errorf("part %d: a location report is all LINES — PartLine's own comment says so — got kind %d", i, p.Kind)
		}
		if p.Text != segs[i].Text {
			t.Errorf("part %d: the words must survive the translation; got %q want %q", i, p.Text, segs[i].Text)
		}
	}
	// A LOCATION REPORT OPENS WITH NO TONE. A tone is a promise of a hazard,
	// and the rotation is the programme — sounding one before an ordinary
	// report would teach a listener to ignore the one that matters.
	if got.Tone != "" {
		t.Errorf("the rotation opens with no attention tone; got %q", got.Tone)
	}
}

func TestAnEmptyRotationComposesAnEmptyScript(t *testing.T) {
	// A card takes the air with its words already on it, and the state
	// machine refuses one that has none. An empty compose must therefore
	// produce an EMPTY script the executor can decline — not a script of one
	// blank line that would read as silence on air.
	if got := scriptFromSegments(nil); !got.Empty() {
		t.Errorf("no segments must yield an empty script the executor can decline; got %+v", got)
	}
}

func TestBlankSegmentsAreDroppedNotSpoken(t *testing.T) {
	got := scriptFromSegments([]synth.Segment{
		{Key: "a", Text: "  "},
		{Key: "b", Text: "Real words."},
	})
	if len(got.Parts) != 1 || got.Parts[0].Text != "Real words." {
		t.Errorf("a blank segment is dead air with a callout already promised; it must be dropped, got %+v", got.Parts)
	}
}
