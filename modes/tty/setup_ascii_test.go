package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// THE SUGGESTION LIST DEGRADES UNDER --ascii, LIKE EVERY OTHER LIST.
//
// TestSetupGoldenASCII already scans the Settings window for "›" and passed
// while the location suggestions drew one — because its fixture focuses a cast
// row, and the suggestion list renders only when the LOCATION row has the focus
// and something has been typed. A gate whose fixture cannot reach the branch is
// vacuous on that branch, which is the same shape as a burst fixture that never
// reaches the head.
//
// So this drives the keys: open Settings, type, and look at what is drawn.
func TestSetupSuggestionsCarryNoNonASCIIUnderASCII(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	h := &setupHarness{}
	cfg := h.config()
	cfg.ASCII = true
	m, err := NewDashboard(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")

	view := stripANSITest(model.(Dashboard).View().Content)
	// THE FIXTURE IS ASSERTED VALID FIRST: without a suggestion on screen this
	// test scans a window that never drew the line it is named for.
	if !strings.Contains(view, "Oceanside, CA (92057)") {
		t.Fatalf("no suggestion rendered, so this pins nothing:\n%s", view)
	}
	for _, glyph := range []string{"›", "▾", "▲", "▼", "●", "○", "—"} {
		if strings.Contains(view, glyph) {
			t.Errorf("--ascii Settings still carries %q — it needs an ASCII form from the glyph set", glyph)
		}
	}
}
