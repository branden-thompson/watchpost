package tty

import (
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
