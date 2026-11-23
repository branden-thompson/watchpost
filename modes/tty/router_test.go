package tty

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// P0's whole claim is that NOTHING CHANGES. These pin it at the ROUTER, which
// is the subject the existing frame goldens do not cover: they call
// Dashboard.View() directly, so a Router that rendered the wrong surface — or
// nothing at all — would leave every one of them green.

func TestRouterRendersObserverByteForByte(t *testing.T) {
	d := goldenDash(t, false)
	want := d.View().Content
	got := NewRouter(d).View().Content
	if got != want {
		t.Errorf("the Router must render Observer unchanged (P0: nothing changes)\n"+
			"  router frame: %d bytes\n  observer frame: %d bytes\n  first difference at %d",
			len(got), len(want), firstDiff(got, want))
	}
}

func TestRouterStartsOnObserver(t *testing.T) {
	if got := NewRouter(goldenDash(t, false)).active; got != SurfaceObserver {
		t.Errorf("a station comes up on Observer; active = %d, want %d", got, SurfaceObserver)
	}
}

func TestRouterDelegatesInitToTheActiveSurface(t *testing.T) {
	d := goldenDash(t, false)
	if NewRouter(d).Init() == nil && d.Init() != nil {
		t.Error("the Router must delegate Init: Observer asks for its background colour and the Router dropped it")
	}
}

func TestRouterPassesAKeyToTheActiveSurface(t *testing.T) {
	d := goldenDash(t, false)
	var r tea.Model = NewRouter(d)
	r, _ = r.Update(tea.KeyPressMsg{Code: '?'})
	if !strings.Contains(r.View().Content, "Help") {
		t.Error("a keypress must reach the active surface: '?' opened no help window through the Router")
	}
}

func firstDiff(a, b string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

// P1: the fan-out is now OBSERVABLE, which is why FR-1.2 waited for a second
// surface. With one surface a fan-out and a delegation are indistinguishable.

func TestSizeReachesTheINACTIVESurface(t *testing.T) {
	var m tea.Model = NewRouter(goldenDash(t, false))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 171, Height: 61})
	r := m.(Router)
	// THE HEIGHT AS DRAWN: the console's frame is clamped to it, a row a line
	// (0.19.6 D-22).
	if h := drawnRows(r.broadcaster.View().Content); r.broadcaster.Width() != 171 || h != 61 {
		t.Errorf("a resize must reach BOTH surfaces; the inactive one has %dx%d, want 171x61 "+
			"(an inactive surface with a stale size renders wrong the instant it is swapped to)",
			r.broadcaster.Width(), h)
	}
}

func TestBackgroundColourReachesTheINACTIVESurface(t *testing.T) {
	// DRAWN (0.19.6 D-22). Every theme grounds the window in one colour whatever
	// the terminal reports (platform/render/themes.go fills WindowBGLight from
	// WindowBGDark), so the console draws the terminal's darkness in its UP
	// NEXT box's ground, under a theme whose modal grounds differ between a
	// dark terminal and a light one.
	var m tea.Model = NewRouter(goldenDash(t, false))
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
	if !render.SetTheme("Synthwave '84") {
		t.Fatal("no Synthwave '84 theme")
	}
	t.Cleanup(func() { render.SetTheme(render.DefaultThemeName) })
	_, darkGround := render.ModalTone(true)
	if _, lightGround := render.ModalTone(false); darkGround == lightGround {
		t.Fatal("the theme grounds a dark terminal's window as a light one's; this test proves nothing")
	}
	m, _ = m.Update(tea.WindowSizeMsg{Width: consoleW, Height: consoleH})
	m, _ = m.Update(LineupMsg{Lineup: mainTrackOf(t, ttytest.Card(t, "a", "Oceanside, CA"))})
	light := m.(Router).broadcaster.View().Content
	m, _ = m.Update(tea.BackgroundColorMsg{Color: color.Black})
	if r := m.(Router); r.broadcaster.View().Content == light {
		t.Error("the background colour must reach BOTH surfaces; the inactive one still thinks the terminal is light")
	}
}

func TestObserverStillGetsItsOwnSizeToo(t *testing.T) {
	var m tea.Model = NewRouter(goldenDash(t, false))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 171, Height: 61})
	if r := m.(Router); r.observer.width != 171 {
		t.Errorf("fanning out must not stop the ACTIVE surface receiving it; observer width = %d, want 171", r.observer.width)
	}
}

// The console must inherit Observer's --ascii decision: one setting, one
// carrier. Two surfaces disagreeing about whether a terminal can draw a glyph
// is the two-carriers shape this codebase has removed twice.
//
// AS DRAWN (0.19.6 D-57): built --ascii, the console's whole frame holds no
// rune past ASCII; built without it, the same frame holds at least one, so the
// check cannot pass on a frame that draws no glyph at all.
func TestTheConsoleInheritsTheASCIIDecision(t *testing.T) {
	for _, ascii := range []bool{true, false} {
		var m tea.Model = NewRouter(goldenDash(t, ascii))
		m, _ = m.Update(tea.WindowSizeMsg{Width: consoleW, Height: consoleH})
		r := m.(Router)
		r.active = SurfaceBroadcaster
		wide := strings.IndexFunc(ttytest.StripANSI(r.View().Content), func(c rune) bool { return c > 0x7F }) >= 0
		if ascii && wide {
			t.Error("Observer was built --ascii and the console was not; one setting must not have two carriers")
		}
		if !ascii && !wide {
			t.Fatal("the console built without --ascii draws no glyph past ASCII; this test proves nothing")
		}
	}
}

// The console's OWN messages reach it even while it is inactive.
//
// A THIRD ROUTING CATEGORY, and it exists for the same reason the fan-out
// does: a surface that only learns things while on screen is stale the
// instant the operator swaps to it — and here the stale thing would be the
// running order and whether the station is on the air.
func TestTheConsoleGetsItsOwnMessagesWhileInactive(t *testing.T) {
	var m tea.Model = NewRouter(goldenDash(t, false)) // Observer active
	m, _ = m.Update(StationMsg{Power: lineup.OffAir})
	if got := m.(Router).broadcaster.Power(); got != lineup.OffAir {
		t.Errorf("the console must learn the station's state while inactive; got %v", got)
	}
}
