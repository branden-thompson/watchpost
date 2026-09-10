package tty

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// broadcaster_cardrow_test.go — D-52's anchoring rule, enforced rather than drawn.
//
// THE MOCK IS A RULE, NOT A PICTURE (HUM LEAD, 2026-09-10: "we don't want to
// hard code geometry"). What the card owes at every width:
//
//	title    centred, truncates KIND-FIRST so the subject survives
//	badge    right-anchored, inboard of the handle
//	handle   right-most and fixed — it is the ADDRESS the operator types, so it
//	         is the one thing that never truncates and never moves
//
// A HAND-ROLLED VERSION OF THIS HAD A REAL BUG, found while generating the mock:
// it tested whether the title FIT BY LENGTH, but a centred title can be short
// enough to fit and still run through the badge. The go-studs row reserves the
// badge's width when it sizes the fill column, so the collision is structurally
// impossible rather than policed.

func aCard(t *testing.T, headline string) lineup.Card {
	t.Helper()
	c, err := lineup.Propose(lineup.Card{ID: "c", Slot: lineup.LocationReport,
		Subject: "oceanside", Headline: headline})
	if err != nil {
		t.Fatalf("proposing: %v", err)
	}
	return c
}

func TestTheCardRowFillsTheLaneItIsGiven(t *testing.T) {
	c := aCard(t, "LOCATION REPORT • OCEANSIDE, CA 92057")
	for _, lane := range []int{98, 118, 128, 148} {
		got := newCardLane(lane, render.Opts{ASCII: true}.Glyphs()).render(c, "6", "STANDARD")
		if w := utf8.RuneCountInString(got); w != lane {
			t.Errorf("lane %d: the row must fill it exactly; got %d cells", lane, w)
		}
	}
}

func TestTheHandleIsRightMostAndNeverMoves(t *testing.T) {
	// It is what the operator types to address the card, so a handle that
	// shifted with the headline would move the target between renders.
	for _, lane := range []int{98, 128, 148} {
		for _, headline := range []string{
			"OCEANSIDE, CA",
			"LOCATION REPORT • OCEANSIDE, CA 92057",
			"LOCATION REPORT • RANCHO SANTA MARGARITA, CA 92688 (COASTAL AND VALLEY AREAS)",
		} {
			got := newCardLane(lane, render.Opts{ASCII: true}.Glyphs()).render(aCard(t, headline), "6", "STANDARD")
			if !strings.HasSuffix(strings.TrimRight(got, " "), chipFor("6")) {
				t.Fatalf("lane %d, %q: the handle must be right-most; got %q", lane, headline, got)
			}
		}
	}
}

func TestALongHeadlineNeverReachesTheBadge(t *testing.T) {
	// THE BUG THIS EXISTS FOR. A centred title that fits by LENGTH can still
	// overrun the badge by POSITION — the hand-rolled draft produced
	// "…(COASTAL)D•", eating the badge from the left.
	long := "LOCATION REPORT • RANCHO SANTA MARGARITA, CA 92688 (COASTAL AND VALLEY AREAS AND BEYOND)"
	for _, lane := range []int{98, 108, 128, 148} {
		got := newCardLane(lane, render.Opts{ASCII: true}.Glyphs()).render(aCard(t, long), "6", "STANDARD")
		i := strings.Index(got, "STANDARD")
		if i < 1 {
			t.Fatalf("lane %d: the badge must survive; got %q", lane, got)
		}
		// The cell before the badge's own delimiter is blank: nothing has run
		// into it.
		if got[i-1] != ' ' && got[i-1] != '*' {
			t.Errorf("lane %d: the headline reached the badge; got %q", lane, got)
		}
		if w := utf8.RuneCountInString(got); w != lane {
			t.Errorf("lane %d: still exactly the lane; got %d", lane, w)
		}
	}
}

func TestANarrowLaneKeepsTheSubjectAndDropsTheKind(t *testing.T) {
	// KIND-FIRST, SUBJECT-LAST. The kind repeats down the whole lane and carries
	// almost nothing there; the subject is the only part that says WHICH card
	// this is, so it is what survives.
	//
	// The lane is chosen so the drop actually happens — an earlier version of
	// this test picked a width where nothing was dropped and SKIPPED, which is
	// not a test.
	got := newCardLane(56, render.Opts{ASCII: true}.Glyphs()).
		render(aCard(t, "LOCATION REPORT • OCEANSIDE, CA 92057"), "6", "STANDARD")
	if !strings.Contains(got, "OCEANSIDE") {
		t.Errorf("the subject identifies the card and must survive; got %q", got)
	}
	if strings.Contains(got, "LOCATION REPORT") {
		t.Errorf("the kind is what gets dropped first; got %q", got)
	}
}

// THE TITLE IS CENTRED, and nothing asserted it — a plant that switched the
// column to left alignment changed no test. It is the one thing about the card
// that a reader notices immediately and a test never would.
func TestTheHeadlineIsCentredInWhatIsLeftOfTheLane(t *testing.T) {
	got := newCardLane(148, render.Opts{ASCII: true}.Glyphs()).
		render(aCard(t, "OCEANSIDE, CA"), "6", "STANDARD")
	i := strings.Index(got, "OCEANSIDE, CA")
	if i < 0 {
		t.Fatalf("the headline must be drawn; got %q", got)
	}
	// It is centred in the room the badge leaves, so there is real space on BOTH
	// sides. Left-aligned would put it at or near column 0.
	if i < 10 {
		t.Errorf("the headline is centred, not flush left; it starts at column %d in %q", i, got)
	}
	after := strings.Index(got, "*STANDARD*") - (i + len("OCEANSIDE, CA"))
	if after < 5 {
		t.Errorf("centred means space on the right too; only %d cells before the badge", after)
	}
}

// FR-7.3: SILENT OVERFLOW IS A DEFECT, NOT A DEGRADATION. The row comes from a
// third-party component whose contract this package does not own, so its output
// is clamped to the lane rather than trusted to fit — F-55 measured the other
// surface rendering 57 cells into a 20-cell terminal.
func TestAnAbsurdlyNarrowLaneStillNeverOverflows(t *testing.T) {
	for _, lane := range []int{1, 4, 12, 20, 30} {
		got := newCardLane(lane, render.Opts{ASCII: true}.Glyphs()).
			render(aCard(t, "LOCATION REPORT • OCEANSIDE, CA 92057"), "6", "STANDARD")
		if w := utf8.RuneCountInString(got); w > lane {
			t.Errorf("lane %d: rendered %d cells — %q", lane, w, got)
		}
	}
}

func TestTheBadgeSitsBetweenTheHeadlineAndTheHandle(t *testing.T) {
	got := newCardLane(148, render.Opts{ASCII: true}.Glyphs()).render(aCard(t, "OCEANSIDE, CA"), "T", "PRIORITY")
	b, h := strings.Index(got, "PRIORITY"), strings.Index(got, chipFor("T"))
	if b < 0 || h < 0 {
		t.Fatalf("both must be present; got %q", got)
	}
	if b >= h {
		t.Errorf("the badge is inboard of the handle; got %q", got)
	}
}

// D-55: A FABRICATED TAKEOVER SAYS SO IN THE LINE-UP TOO.
//
// Every surface that existed when the injector's safeguards were written marks
// one — the band at both ends, the severe window leading its EVENT column, the
// audio four times over. The console did not, because it did not exist yet, and
// `Card` did not carry the fact at all.
//
// THE MARK IS A PREFIX, NOT PART OF THE HEADLINE, so the row reserves its width
// and the title's truncation can never eat it. A HALF-EATEN MARK IS WORSE THAN
// NONE: "**TEST EV…" beside a tornado warning reads as damage, not as a label.
func TestAFabricatedCardIsMarkedInTheLineUp(t *testing.T) {
	c := aCard(t, "TORNADO WARNING + 2 more")
	c.Test = true
	got := newCardLane(148, render.Opts{ASCII: true}.Glyphs()).render(c, "T", "PRIORITY")

	if !strings.Contains(got, testEventMark) {
		t.Errorf("a fabricated takeover must say so where the operator reads the line-up; got %q", got)
	}
	// IT LEADS. A mark after the headline is a mark the eye reaches second, and
	// the headline is the part that looks like a real warning.
	if i, j := strings.Index(got, testEventMark), strings.Index(got, "TORNADO"); i < 0 || j < 0 || i > j {
		t.Errorf("the mark leads the card; got %q", got)
	}
}

func TestARealCardCarriesNoMark(t *testing.T) {
	// THE HALF THAT MATTERS MOST: a mark on a real hazard teaches an operator to
	// ignore the mark.
	got := newCardLane(148, render.Opts{ASCII: true}.Glyphs()).
		render(aCard(t, "TORNADO WARNING + 2 more"), "T", "PRIORITY")
	if strings.Contains(got, testEventMark) {
		t.Errorf("a real hazard is never marked as a test; got %q", got)
	}
}

func TestTheTestMarkSurvivesTheNarrowestLane(t *testing.T) {
	// The mark is the one thing on the card that must never be truncated away,
	// so it is asserted at every supported width AND below them.
	c := aCard(t, "TORNADO WARNING FOR SAN DIEGO COUNTY COASTAL AREAS + 4 more")
	c.Test = true
	for _, lane := range []int{98, 108, 128, 148} {
		got := newCardLane(lane, render.Opts{ASCII: true}.Glyphs()).render(c, "T", "PRIORITY")
		if !strings.Contains(got, testEventMark) {
			t.Errorf("lane %d: the mark was truncated away; got %q", lane, got)
		}
		if w := utf8.RuneCountInString(got); w != lane {
			t.Errorf("lane %d: and it still fills the lane exactly; got %d", lane, w)
		}
	}
}

// A PARTIAL MARK IS THE ONE OUTPUT THAT IS WORSE THAN NO MARK. "**TEST EV…"
// beside a tornado warning reads as rendering damage, not as a label — and an
// operator who reads it as damage reads the warning as real.
//
// So at any width the row must show the mark WHOLE or not at all. This is
// asserted below the supported floor as well as at it, because the rule is
// about the mark and not about the breakpoints.
func TestTheTestMarkIsNeverShownPartially(t *testing.T) {
	c := aCard(t, "TORNADO WARNING FOR SAN DIEGO COUNTY + 4 more")
	c.Test = true
	for _, lane := range []int{4, 8, 12, 16, 20, 30, 60, 98, 148} {
		got := newCardLane(lane, render.Opts{ASCII: true}.Glyphs()).render(c, "T", "PRIORITY")
		whole := strings.Contains(got, testEventMark)
		partial := !whole && strings.Contains(got, "TEST")
		if partial {
			t.Errorf("lane %d: the mark is shown in pieces — %q", lane, got)
		}
		if w := utf8.RuneCountInString(got); w > lane {
			t.Errorf("lane %d: and it overflowed to %d cells", lane, w)
		}
	}
}

// AT A NARROW-BUT-DRAWABLE WIDTH THE CARD IS STILL DRAWN, AND STILL MARKED.
//
// The two mechanisms that keep the mark whole are different in scope and this
// is what separates them: the column is NON-TRUNCATABLE so the component never
// shortens it, and the row is DROPPED if the mark is missing so nothing else
// can. Without the first, a squeezed lane truncates the mark, the second fires,
// and the card VANISHES — a fabricated takeover that silently disappears from
// the line-up instead of being labelled.
//
// These widths are below the supported floor on purpose: they are where the
// squeeze happens, and the rule is about the mark rather than the breakpoints.
func TestANarrowLaneStillDrawsTheMarkedCardRatherThanDroppingIt(t *testing.T) {
	c := aCard(t, "TORNADO WARNING FOR SAN DIEGO COUNTY + 4 more")
	c.Test = true
	for _, lane := range []int{36, 44, 52, 60, 80} {
		got := newCardLane(lane, render.Opts{ASCII: true}.Glyphs()).render(c, "T", "PRIORITY")
		if got == "" {
			t.Errorf("lane %d: the card was dropped; it is narrow, not unlabellable", lane)
			continue
		}
		if !strings.Contains(got, testEventMark) {
			t.Errorf("lane %d: drawn without its mark — %q", lane, got)
		}
	}
}

// chipFor is how a handle actually renders — the CHIP, not the brackets a mock
// draws around it. Asserting the literal "[ 6 ]" encoded the costume: KeyCap
// paints " 6 " with the chip background in colour and falls back to "[6]"
// without it, so the literal was only ever right in one mode.
func chipFor(handle string) string { return render.Opts{ASCII: true}.KeyCap(handle) }
