package tty

// Quality pass Q3 (plan §2.5 tick predicate; PF-2, R2-23, BQ-12): the
// shimmer tick runs only while a frame would change on its own, and every
// wall-clock element it serves is pinned here so the predicate cannot
// silently drop one.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// tickOf runs the model through Update and reports whether a tick was
// armed (the command is non-nil and, when a batch, carries a tick).
func armedAfter(t *testing.T, m tea.Model, msg tea.Msg) (tea.Model, bool) {
	t.Helper()
	next, cmd := m.Update(msg)
	return next, next.(Dashboard).tickArmed && cmd != nil
}

func TestTickArmedOnlyWhileAnimating(t *testing.T) {
	loaded := snap() // observation + daily present: nothing loads
	loading := snap()
	loading.Locations[0].Daily = nil // the shimmer row

	rows := []struct {
		name  string
		setup func(d Dashboard) Dashboard
		want  bool
	}{
		{"idle: nothing loads, radio off, no modal", func(d Dashboard) Dashboard { return d }, false},
		{"a loading row keeps the shimmer running", func(d Dashboard) Dashboard { d.snap = loading; return d }, true},
		{"a pending volume blink", func(d Dashboard) Dashboard {
			d.volFlash, d.volFlashEnd = "+", time.Now().Add(350*time.Millisecond)
			return d
		}, true},
		{"the marquee (playing, narration, viz off)", func(d Dashboard) Dashboard {
			d.radioPlaying, d.radioState, d.radioDetail = true, "playing", "Tonight. Mostly clear."
			return d
		}, true},
		{"the marquee is the viz tick's while it runs", func(d Dashboard) Dashboard {
			d.radioPlaying, d.radioState, d.radioDetail, d.vizTicking = true, "playing", "Tonight. Mostly clear.", true
			return d
		}, false},
		{"LIVE RADIO has no marquee", func(d Dashboard) Dashboard {
			d.radioPlaying, d.radioState, d.radioDetail, d.radioLive = true, "playing", "KEC80", true
			return d
		}, false},
		{"the min player has no marquee", func(d Dashboard) Dashboard {
			d.radioPlaying, d.radioState, d.radioDetail, d.radioMin = true, "playing", "Tonight.", true
			return d
		}, false},
		{"[S] ages", func(d Dashboard) Dashboard { d.showStatus = true; return d }, true},
		{"Details labels and LoadingDots", func(d Dashboard) Dashboard { d.showDetails = true; return d }, true},
	}
	for _, row := range rows {
		d := dash(t).(Dashboard)
		d.snap = loaded
		d = row.setup(d)
		if got := d.tickNeeded(); got != row.want {
			t.Errorf("%s: tickNeeded = %v, want %v", row.name, got, row.want)
		}
		// A tick message re-arms only while the predicate holds.
		_, armed := armedAfter(t, d, tickMsg{})
		if armed != row.want {
			t.Errorf("%s: after a tick, armed = %v, want %v", row.name, armed, row.want)
		}
	}
}

func TestTickRearmsOnTheTransitionAndNeverTwice(t *testing.T) {
	m := dash(t)
	if m.(Dashboard).tickArmed {
		t.Fatal("a loaded snapshot with the radio off arms nothing")
	}
	loading := snap()
	loading.Locations[0].Daily = nil
	m, armed := armedAfter(t, m, SnapshotMsg{Snap: loading})
	if !armed {
		t.Fatal("a snapshot with a loading row arms the tick")
	}
	// A second message while the tick is in flight must not arm another.
	next, cmd := m.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	if cmd != nil {
		t.Fatal("a tick already in flight: no second one (never two tickers)")
	}
	// The data lands: the tick that fires next is the last one.
	next, _ = next.Update(SnapshotMsg{Snap: snap()})
	_, cmd = next.Update(tickMsg{})
	if cmd != nil {
		t.Fatal("nothing animates after the row loaded: the tick stops")
	}
}

func TestVolumeBlinkClearsOnTheTickAfterItExpires(t *testing.T) {
	d := dash(t).(Dashboard)
	d.volFlash, d.volFlashEnd = "+", time.Now().Add(-time.Millisecond) // already expired
	m, cmd := d.Update(tickMsg{})
	if nd := m.(Dashboard); nd.volFlash != "" || cmd != nil {
		t.Fatalf("the expired blink clears on the tick and the tick stops: flash=%q cmd=%v", nd.volFlash, cmd != nil)
	}
	d.volFlashEnd = time.Now().Add(time.Second) // still pending: the tick keeps running
	if m, cmd := d.Update(tickMsg{}); m.(Dashboard).volFlash != "+" || cmd == nil {
		t.Fatal("a pending blink stays and keeps the tick armed")
	}
}

func TestMarqueeAdvancesAcrossTicksWithVizOff(t *testing.T) {
	d := dash(t).(Dashboard)
	d.radioPlaying, d.radioState, d.radioSpoken = true, "playing", 4*time.Second
	d.radioDetail = strings.Repeat("The forecast for tonight is mostly clear with a light wind. ", 4)
	d.radioSince = time.Now()
	first := stripANSITest(d.View().Content)
	d.radioSince = time.Now().Add(-2 * time.Second) // two seconds of speech later
	later := stripANSITest(d.View().Content)
	if first == later {
		t.Fatal("the marquee window must move with the voice between two frames (UAT 83)")
	}
}

func TestStatusAgesAndDetailsLabelsMoveWithTheClock(t *testing.T) {
	d := dash(t).(Dashboard)
	old := snap()
	old.Providers[0].FetchedAt = time.Now().Add(-90 * time.Second)
	d.snap = old
	d.showStatus = true
	if v := stripANSITest(d.View().Content); !strings.Contains(v, "fetched") {
		t.Fatalf("the [S] modal shows provider ages:\n%s", v)
	}
	d.showStatus, d.showDetails = false, true
	loc := &d.snap.Locations[0]
	loc.Fire.AsOf = time.Now()
	loc.Fire.Hotspots = []snapshot.Hotspot{{Lat: 33.2, Lon: -117.4, DetectedAt: time.Now().Add(-3 * time.Hour), Confidence: "high"}}
	at3h := stripANSITest(d.View().Content)
	loc.Fire.Hotspots[0].DetectedAt = time.Now().Add(-5 * time.Hour)
	if at5h := stripANSITest(d.View().Content); at3h == at5h {
		t.Fatal("Details carries time-relative labels (a hotspot's age), which is why showDetails is a predicate term")
	}
}
