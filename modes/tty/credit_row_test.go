package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// A CREDIT ROW IS ONE COMPONENT (D-235): a source's chip, a dash, what it
// gives, its endpoint at the right margin light blue, and - only where it has
// one - a note under the summary. Every row of a set has its dash in one
// column, whatever its chip's width, so a source added later lines up.
func TestACreditRowIsOneComponent(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	badgeW := badgeWidth([]CreditLine{{Badge: "NWS"}, {Badge: "GEONAMES"}})
	plain := creditRows(CreditLine{Badge: "NWS", What: "National Weather Service", Host: "api.weather.gov"}, 70, badgeW)
	if len(plain) != 1 {
		t.Fatalf("a row with no note is %d lines; want one", len(plain))
	}
	row := plain[0]
	if !strings.Contains(row, chipFace("NWS")) || !strings.HasSuffix(row, render.Tint("api.weather.gov", render.Tok(render.AboutHost))) || render.Width(row) != 70 {
		t.Errorf("the row is %q (%d cells); want NWS's chip, the host light blue at the margin, 70 cells", row, render.Width(row))
	}
	noted := creditRows(CreditLine{Badge: "GEONAMES", What: "Cities & Postal Codes", Host: "geonames.org", Note: "CC BY 4.0"}, 70, badgeW)
	if len(noted) != 2 {
		t.Fatalf("a row with a note is %d lines; want two", len(noted))
	}
	dash := func(l string) int { return strings.Index(stripANSITest(l), " - ") }
	if dash(row) != dash(noted[0]) || dash(row) < 0 {
		t.Errorf("the dashes stand at %d and %d; want one column", dash(row), dash(noted[0]))
	}
	if !strings.Contains(noted[1], render.Tint("CC BY 4.0", "3;"+render.Tok(render.AboutNote))) {
		t.Errorf("the note is %q; want it italic in the note's tone (D-236)", noted[1])
	}
	summaryAt := strings.Index(stripANSITest(noted[0]), "Cities")
	if note := stripANSITest(noted[1]); strings.Index(note, "CC BY 4.0") != summaryAt || strings.TrimSpace(note) != "CC BY 4.0" {
		t.Errorf("the note is %q; want it under the summary, at column %d", note, summaryAt)
	}
	long := creditRows(CreditLine{Badge: "NWS", What: strings.Repeat("a long summary ", 4), Host: "api.weather.gov"}, 70, badgeW)
	if len(long) != 2 || !strings.HasSuffix(long[1], render.Tint("api.weather.gov", render.Tok(render.AboutHost))) {
		t.Errorf("a summary too long for its host puts the host on a line of its own at the margin: %q", long)
	}
}

// EVERY SOURCE HAS ITS CHIP (D-235, D-134): the seven new ones among them.
func TestTheNewSourcesHaveTheirChips(t *testing.T) {
	for _, name := range []string{"NHC", "NWR", "FIRMS", "RELAYS", "GEONAMES", "OFM", "PIPER"} {
		if !ChipKnown(name) {
			t.Errorf("%s has no chip of its own", name)
		}
	}
}
