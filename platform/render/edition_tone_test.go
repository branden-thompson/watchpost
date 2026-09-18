package render

// edition_tone_test.go — the two editions do not wear the same word colour.
//
// HUM LEAD, 2026-09-14: "Let's make the 'Broadcaster' text in the mastHead
// Orange vs. the Bold Light Blue - so: Observer - Bold Light Blue /
// Broadcaster - Bold Orange."

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// monochromeExceptions names the themes where the two editions MAY share a
// tone, with the reason. A theme added later that happens to leave the new
// token unset fails below rather than passing quietly.
var monochromeExceptions = map[string]string{
	"Monochrome": "the theme has no orange, and inventing one is the thing it exists to refuse; the WORD still differs",
}

func TestEachEditionWearsItsOwnToneInEveryTheme(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	for _, name := range ThemeNames() {
		if !SetTheme(name) {
			t.Fatalf("%s did not load", name)
		}
		obs, bc := Tok(TitleEdition), Tok(TitleEditionBroadcaster)
		if bc == "" {
			t.Errorf("%s: the console's edition word has no tone at all; it would draw unstyled", name)
			continue
		}
		// BOLD IN BOTH, which is the half the ruling keeps: "Bold Light Blue"
		// and "Bold Orange".
		for tok, v := range map[Token]string{TitleEdition: obs, TitleEditionBroadcaster: bc} {
			if !strings.HasPrefix(v, "1;") {
				t.Errorf("%s %s: %q is not bold", name, tok, v)
			}
		}
		if why, excused := monochromeExceptions[name]; excused {
			t.Logf("%s: editions share a tone — %s", name, why)
			continue
		}
		if obs == bc {
			t.Errorf("%s: both editions wear %q, so the masthead distinguishes them by WORD alone — "+
				"which is the thing the ruling changed", name, obs)
		}
	}
}

// AND THE WORDMARK ACTUALLY USES IT. A token nothing reads is a colour nobody
// sees, and the two call sites (the masthead and the About window) both go
// through Wordmark precisely so they cannot disagree.
func TestTheWordmarkPaintsTheConsolesEditionWithItsOwnToken(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	// THROUGH Tint, NOT AGAINST THE RAW TOKEN. `Tint` expands "1;208" into
	// "1;38;5;208" — a 256-colour index, not a literal SGR — so a Contains
	// against the table's value fails on a wordmark that is perfectly correct.
	// Building the expected segment the same way the painter does is the only
	// comparison that measures the tone rather than the encoding.
	bc, obs := Wordmark(EditionBroadcaster), Wordmark(EditionObserver)
	if want := Tint(EditionBroadcaster, Tok(TitleEditionBroadcaster)); !strings.Contains(bc, want) {
		t.Errorf("the console's wordmark does not wear its own tone.\n  want segment: %q\n  got:          %q", want, bc)
	}
	if want := Tint(EditionObserver, Tok(TitleEdition)); !strings.Contains(obs, want) {
		t.Errorf("the listener's wordmark lost its own tone.\n  want segment: %q\n  got:          %q", want, obs)
	}
	if strings.Contains(obs, Tint(EditionObserver, Tok(TitleEditionBroadcaster))) {
		t.Error("Observer is wearing the console's tone: the two editions are not being told apart")
	}
}
