package tty

// broadcaster_manifest_test.go — a card as a MANIFEST (D-87).
//
// HUM LEAD, 2026-09-11: "What DOES MAKE SENSE for the operator is to get a
// SUMMARY of what the report contains *before* it goes on air."
//
// So the card answers three questions and stops: what is it doing, how fresh is
// what it will say, and what is in it.

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// A CARD SAYS WHAT IT IS DOING, and the three answers are three different
// things the operator may do next.
func TestACardSaysWhatItIsDoing(t *testing.T) {
	for _, tc := range []struct {
		state lineup.State
		want  string
	}{
		{lineup.OnAir, "READING"},
		{lineup.Standby, "Ready for Read-Out"},
		{lineup.Admitted, "Scheduled; Awaiting Data"},
	} {
		got := cardStatus(lineup.Card{State: tc.state}, true)
		if !strings.Contains(got, tc.want) {
			t.Errorf("a card at %v says %q; want it to say %q", tc.state, got, tc.want)
		}
	}

	// THE LIVE CARD SAYS WHAT THE OPERATOR MAY STILL DO, which are two rules
	// they would otherwise have had to read a document to know: a card on the
	// air is locked to management (D-45) and can still be taken over (D-82).
	live := cardStatus(lineup.Card{State: lineup.OnAir}, true)
	for _, want := range []string{"Management Locked", "Taken-Over"} {
		if !strings.Contains(live, want) {
			t.Errorf("the live card's status is %q; it does not mention %q", live, want)
		}
	}

	// AND AN EMPTY SLOT SAYS NOTHING. A status on a slot with no card in it is a
	// claim about a card that is not there.
	if got := cardStatus(lineup.Card{}, false); got != "" {
		t.Errorf("an empty slot claims a status: %q", got)
	}
}

// THE THREE ANSWERS ARE DIFFERENT ANSWERS. Three states that resolved to one
// sentence would pass every check above and tell the operator nothing.
func TestTheThreeStatusesAreToldApart(t *testing.T) {
	seen := map[string]lineup.State{}
	for _, st := range []lineup.State{lineup.OnAir, lineup.Standby, lineup.Admitted} {
		got := cardStatus(lineup.Card{State: st}, true)
		if other, dup := seen[got]; dup {
			t.Errorf("%v and %v both read %q", other, st, got)
		}
		seen[got] = st
	}
}

// HOW FRESH IT IS, AND WHEN IT WAS PULLED — both, because they answer different
// questions: the stamp says which cycle this is, the age says whether to trust
// it before putting it on the air.
func TestACardSaysHowFreshItsDataIs(t *testing.T) {
	now := time.Date(2026, 9, 11, 23, 59, 59, 0, time.UTC)
	c := lineup.Card{BuiltAt: now.Add(-2 * time.Minute)}
	o := render.Opts{ASCII: true}

	got := cardPulled(o, c, func() time.Time { return now }, 70)
	if !strings.Contains(got, "23:57:59") {
		t.Errorf("the stamp is missing from %q", got)
	}
	if !strings.Contains(got, "(2 MIN AGO)") {
		t.Errorf("the age is missing from %q", got)
	}
	// THE AGE IS ANCHORED RIGHT, so the one number the operator sweeps the list
	// for sits in the same column on every card.
	if !strings.HasSuffix(got, "(2 MIN AGO)") {
		t.Errorf("the age is not anchored to the right of the card: %q", got)
	}
	if w := render.Width(got); w != 70 {
		t.Errorf("the stamp row is %d cells of a 70-cell card", w)
	}

	// A CARD THAT WAS NEVER BUILT SAYS SO rather than showing an epoch.
	if got := cardPulled(o, lineup.Card{}, func() time.Time { return now }, 70); strings.Contains(got, "0001") {
		t.Errorf("an unbuilt card shows a zero time: %q", got)
	}
}

// THE MANIFEST IS WHAT THE READ CONTAINS, and it is the card's own — not
// recomposed, not guessed.
func TestTheManifestListsTheCardsContents(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.ascii = 150, true
	c := lineup.Card{Contents: []lineup.Content{
		{Name: "NWS Weather Forecast", Detail: "09/11 - 09/17"},
		{Name: "Watchpost Fire Report", Detail: "3 Hotspots / 10 incidents"},
	}}

	rows := b.manifestRows(c, 70)
	if len(rows) != bcReadLines {
		t.Fatalf("the manifest is %d rows whatever it holds; got %d", bcReadLines, len(rows))
	}
	if !strings.Contains(rows[0], "01.") || !strings.Contains(rows[0], "NWS Weather Forecast") {
		t.Errorf("row 0 is %q; want the first source, numbered", rows[0])
	}
	if !strings.Contains(rows[0], "09/11 - 09/17") {
		t.Errorf("row 0 is %q; the source's range is missing", rows[0])
	}
	// THE HEIGHT IS FIXED, like the card's: a manifest that grew with its
	// contents would move every card below it as data arrived.
	if strings.TrimSpace(rows[2]) != "" {
		t.Errorf("a two-source manifest drew something in row 2: %q", rows[2])
	}
	// AND IT NEVER SPILLS. More sources than rows are cut rather than pushing
	// the card's controls off it.
	for i := range 8 {
		c.Contents = append(c.Contents, lineup.Content{Name: "extra", Detail: string(rune('a' + i))})
	}
	if got := len(b.manifestRows(c, 70)); got != bcReadLines {
		t.Errorf("a ten-source manifest drew %d rows, want %d", got, bcReadLines)
	}
}

// THE HEADING NAMES BOTH COLUMNS, and the second is deliberately vague because
// the sources answer different questions.
func TestTheManifestHeadingNamesItsColumns(t *testing.T) {
	got := manifestHeading(70)
	for _, want := range []string{"##.", "NAME", "RANGE / INCIDENTS"} {
		if !strings.Contains(got, want) {
			t.Errorf("the heading is %q; %q is missing", got, want)
		}
	}
	// THE DETAIL COLUMN LINES UP WITH THE ROWS UNDER IT. A heading whose second
	// column sat somewhere else would be worse than none.
	row := manifestRow("01.", "NWS Weather Forecast", "09/11 - 09/17", 70)
	if strings.Index(got, "RANGE") != strings.Index(row, "09/11") {
		t.Errorf("the heading's detail column is at %d and the row's at %d:\n%q\n%q",
			strings.Index(got, "RANGE"), strings.Index(row, "09/11"), got, row)
	}
}

// THE KIND COMES FROM THE SLOT REGISTRY, NOT FROM THE PRODUCER (D-87).
//
// The reference draws "LOCATION REPORT • Oceanside, CA" and the producer
// supplies only the location — as it should, because what a card is ABOUT and
// what KIND of card it is are two facts with two owners. The HUM LEAD saw the
// consequence in UAT: the cards read "Oceanside, CA" with no kind at all.
func TestACardsTitleIsItsKindThenItsSubject(t *testing.T) {
	g := render.Opts{ASCII: true}.Glyphs()
	got := cardTitle(lineup.Card{Slot: lineup.LocationReport, Headline: "Oceanside, CA"}, g)
	if !strings.HasPrefix(got, "LOCATION REPORT") {
		t.Errorf("the title is %q; the kind comes first", got)
	}
	if !strings.HasSuffix(got, "Oceanside, CA") {
		t.Errorf("the title is %q; the subject comes last", got)
	}
	// THE SEPARATOR IS THE GLYPH SET'S, so --ascii needs no special case — the
	// parity gate caught the first draft's literal bullet.
	if !strings.Contains(got, g.Bullet) {
		t.Errorf("the title is %q; its separator is not the glyph set's", got)
	}
	// A HAZARD NAMES ITSELF TOO, from the same registry.
	if got := cardTitle(lineup.Card{Slot: lineup.BreakingAlert, Headline: "Tornado Warning"}, g); !strings.HasPrefix(got, "BREAKING ALERT") {
		t.Errorf("a hazard's title is %q", got)
	}
	// AND A CARD MISSING EITHER HALF KEEPS THE OTHER rather than drawing a
	// separator with air on one side of it.
	if got := cardTitle(lineup.Card{Slot: lineup.LocationReport}, g); strings.Contains(got, g.Bullet) {
		t.Errorf("a card with no subject drew a separator: %q", got)
	}
}

// THE CONSOLE HAS ONE CLOCK, AND IT IS NEVER NIL (D-87).
//
// `Broadcaster.clock` already existed and falls back to `time.Now`; the first
// draft of the stamp read the raw `now` FIELD instead, which production never
// sets — so the age the operator uses to decide whether to trust a report was
// missing in the real app and present in every test. Two carriers of one fact,
// and the tests had the one that worked.
func TestTheConsolesStampAlwaysCarriesAnAge(t *testing.T) {
	b := NewBroadcaster() // no clock injected: production's own state
	b.width, b.ascii = 150, true
	lane := newCardLane(b.cardBoxWidth(), b.opts().Glyphs())
	c := lineup.Card{ID: "a", Slot: lineup.LocationReport, Headline: "Oceanside, CA",
		State: lineup.Standby, BuiltAt: time.Now().Add(-3 * time.Minute)}

	rows := b.readBody(b.opts(), lane, c, "1", true)
	stamp := ""
	for _, r := range rows {
		if strings.Contains(r, "DATA PULLED") {
			stamp = r
		}
	}
	if stamp == "" {
		t.Fatal("the card carries no data stamp")
	}
	if !strings.Contains(stamp, "AGO)") {
		t.Errorf("the stamp is %q; without an age the operator cannot tell whether to trust it", stamp)
	}
}
