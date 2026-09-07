package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// F-41 — THE DECK'S WORDS REACH THE SCREEN.
//
// A voice preview takes seconds (Piper reads its model on every run) and can
// fail. The deck says so — "loading <voice>…", "preview failed: <err>" — and
// those words used to land in d.voiceNote, which the retired [V] chooser drew
// and nothing has drawn since. So `p` was silent while it worked and silent when
// it failed, which is UAT 119's complaint with the feedback removed.
//
// TWO ASSERTIONS, AND THE SECOND IS THE ONE THAT BITES. The text must be in the
// frame; and the frame must CHANGE when the note arrives. Settings is memoised
// on setup.gen, so a note stored without touching the generation would be
// written and never drawn — the still-picture defect that cost three UAT rounds
// (F-30). A test that only checked the string would pass against that.
func TestAVoiceNoteFromTheDeckIsDrawnUnderTheRowThatAskedForIt(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	d := setupGolden(t, 133, 44, false, rowCastAlerts)
	var model tea.Model = d

	// The frame as it stands, before the deck says anything.
	before := stripANSITest(model.(Dashboard).View().Content)
	if !strings.Contains(before, "Alerts / Takeovers") {
		t.Fatalf("the fixture is not on the correspondents group:\n%s", before)
	}
	if strings.Contains(before, "loading Karen") {
		t.Fatal("the fixture already shows the note; this would pass without the wire")
	}

	// `p` on the focused row binds the note to it.
	m2, _ := model.(Dashboard).setupPreview()
	model = m2

	// AND THE FRAME IS RENDERED HERE, DELIBERATELY. Without this the memo has
	// nothing cached at the current generation, so the next View is a miss and
	// draws the note whether or not the generation moved — a pin ABOVE the cache
	// cannot see a cache bug (F-30's own lesson, and this test failed to catch
	// its own control until the render was added).
	primed := stripANSITest(model.(Dashboard).View().Content)
	if strings.Contains(primed, "loading Karen") {
		t.Fatal("the note is on screen before the deck said anything")
	}

	model, _ = model.Update(VoiceNoteMsg{Text: "loading Karen…"})
	after := stripANSITest(model.(Dashboard).View().Content)

	if !strings.Contains(after, "loading Karen") {
		t.Errorf("the deck's note never reached the screen — a preview is silent while it works "+
			"and silent when it fails:\n%s", after)
	}
	if after == primed {
		t.Error("the frame did not change when the note arrived: Settings is memoised on setup.gen, " +
			"so a note that does not touch the generation is written and never drawn")
	}
}
