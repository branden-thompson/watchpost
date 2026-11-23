package tty

// console_mute_test.go — the listener's mute, handed across by the Router and drawn
// in the console's station band while it is on air.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// REVIEW 2026-09-17 (ruling 6-ii) — THE LISTENER'S MUTE IS SHOWN ON AIR. [M] in
// the Observer declines every hazard read on a station the operator put ON AIR,
// and the console said nothing. The Router hands the Observer's mute to the
// console, which shows it in the station's own band while ON AIR.
func TestAMutedListenerIsShownWhileTheStationIsOnAir(t *testing.T) {
	// A HELD ALERT ON A STATION THAT WENT TO STANDBY AND THEN ON AIR.
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	held, err := lineup.Lineup{}.Queue(lineup.AlertRail, ttytest.Card(t, "x", "TORNADO WARNING"))
	if err != nil {
		t.Fatal(err)
	}
	b := consoleOf(t, Opening{Now: func() time.Time { return at }}, held)
	b, _ = b.Update(StationMsg{Power: lineup.OffAir})
	b, _ = b.Update(StationMsg{Power: lineup.Running})
	b.SetListenerMuted(true)
	if got := ttytest.StripANSI(b.View().Content); !strings.Contains(got, "LISTENER MUTED") {
		t.Errorf("an ON AIR station with the listener muted shows no LISTENER MUTED:\n%s", got)
	}
	b.SetListenerMuted(false)
	if got := ttytest.StripANSI(b.View().Content); strings.Contains(got, "LISTENER MUTED") {
		t.Error("the mute band persists after the mute is lifted")
	}
	r := routerAt(lineup.Running, SurfaceBroadcaster)
	r.broadcaster, _ = r.broadcaster.Update(tea.WindowSizeMsg{Width: consoleW, Height: consoleH})
	r.observer.tickerMuted = true
	m, _ := r.Update(StationMsg{Power: lineup.Running})
	if !strings.Contains(drawn(m.(Router).broadcaster), "LISTENER MUTED") { // the band says so on air (0.19.6 D-22)
		t.Error("the Router did not hand the Observer's mute to the console")
	}
}
