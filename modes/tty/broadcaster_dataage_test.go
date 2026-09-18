package tty

// broadcaster_dataage_test.go — what the card's two facts are worth at a glance.
//
// HUM LEAD, 2026-09-15: "Let's make 'Ready for Read-Out' Green, and color code
// the other status messages … Color code the (n MIN AGO): < 2 min BLUE, < 5 min
// GREEN, < 10 min YELLOW, < 15 min ORANGE ( we refresh at 15m max so It should
// never get to 'red' )."

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestTheStatusLineIsColourCoded(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	for _, tc := range []struct {
		name  string
		state lineup.State
		want  render.Token
	}{
		{"ready", lineup.Standby, render.ProviderOK},
		{"waiting", lineup.Admitted, render.AlertLabel},
		{"reading", lineup.OnAir, render.ProviderDown},
	} {
		got := cardStatus(lineup.Card{State: tc.state}, true)
		if want := render.Tint("", render.Tok(tc.want)); !strings.HasPrefix(got, want[:strings.Index(want, "m")+1]) {
			t.Errorf("%s: status is not painted with %s:\n  %q", tc.name, tc.want, got)
		}
	}
	// AND THE WAITING STATE IS NOT BOLD — "yellow (non bold)", said in the same
	// breath. A card waiting on a fetch is a caution, not a fault, and weight
	// here would make every unfilled slot shout.
	if got := cardStatus(lineup.Card{State: lineup.Admitted}, true); strings.Contains(got, "\x1b[1m") {
		t.Errorf("the waiting status is bold:\n  %q", got)
	}
}

// THE LADDER, RUNG BY RUNG, AT ITS OWN BOUNDARIES. The interesting values are
// the edges: a test at 1 and 20 minutes would pass on a ladder with any
// thresholds at all.
func TestTheDataAgeLadderTurnsAtItsThresholds(t *testing.T) {
	for _, tc := range []struct {
		age  time.Duration
		want render.Token
	}{
		{0, render.DataNew},
		{119 * time.Second, render.DataNew},
		{2 * time.Minute, render.DataFresh},
		{4*time.Minute + 59*time.Second, render.DataFresh},
		{5 * time.Minute, render.DataUsable},
		{9*time.Minute + 59*time.Second, render.DataUsable},
		{10 * time.Minute, render.DataAged},
		{14*time.Minute + 59*time.Second, render.DataAged},
		{15 * time.Minute, render.DataStale},
		{3 * time.Hour, render.DataStale},
	} {
		if got := dataAgeTone(tc.age); got != tc.want {
			t.Errorf("%v sits on %s, want %s", tc.age, got, tc.want)
		}
	}
}

// AND NO RUNG IS RED. The cadence caps the age at fifteen minutes, so a red one
// would promise the operator a colour they should never meet.
func TestNoRungOfTheLadderIsTheAlarmRed(t *testing.T) {
	t.Cleanup(func() { render.SetTheme(render.DefaultThemeName) })
	for _, name := range render.ThemeNames() {
		if !render.SetTheme(name) {
			t.Fatal(name)
		}
		// MONOCHROME IS EXCUSED BY NAME, WITH ITS REASON: it paints every token
		// the same grey, so "is this rung the alarm red" has no meaning there —
		// the alarm red IS that grey. Excused rather than skipped silently, so a
		// future theme that collapses its palette fails here instead of passing.
		if name == "Monochrome" {
			t.Logf("%s: excused — the theme has one colour, so no rung can differ from any other", name)
			continue
		}
		red := render.Tok(render.ProviderDown)
		for _, tok := range []render.Token{render.DataNew, render.DataFresh, render.DataUsable, render.DataAged, render.DataStale} {
			if render.Tok(tok) == red {
				t.Errorf("%s: %s is the alarm red %q; the ruling says the ladder never reaches it", name, tok, red)
			}
		}
	}
}

// AND THE CARD ACTUALLY PAINTS THE AGE IT COMPUTES. A ladder nothing draws is a
// table of constants.
func TestTheCardPaintsTheAgeWithItsRung(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	built := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return built.Add(7 * time.Minute) } // the YELLOW rung
	got := cardPulledShort(render.Opts{}, lineup.Card{BuiltAt: built}, now)

	if !strings.Contains(got, "7 MIN AGO") {
		t.Fatalf("the card does not say how old the pull is: %q", got)
	}
	want := render.Tint("(7 MIN AGO)", render.Tok(render.DataUsable))
	if !strings.Contains(got, want) {
		t.Errorf("the age is not painted on its rung.\n  want: %q\n  got:  %q", want, got)
	}
}
