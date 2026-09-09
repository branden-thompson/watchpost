package tty

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
	if r.broadcaster.width != 171 || r.broadcaster.height != 61 {
		t.Errorf("a resize must reach BOTH surfaces; the inactive one has %dx%d, want 171x61 "+
			"(an inactive surface with a stale size renders wrong the instant it is swapped to)",
			r.broadcaster.width, r.broadcaster.height)
	}
}

func TestBackgroundColourReachesTheINACTIVESurface(t *testing.T) {
	var m tea.Model = NewRouter(goldenDash(t, false))
	m, _ = m.Update(tea.BackgroundColorMsg{Color: color.Black})
	if r := m.(Router); !r.broadcaster.darkBG {
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
