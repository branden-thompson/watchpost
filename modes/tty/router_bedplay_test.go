package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/modes/tty/console"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// SPACE PLAYS THE BED'S RELAY, AND NEVER OVER THE PROGRAMME (D-215, D-216).
// Choosing a relay tunes nothing, so space is how the operator hears it on
// standby; on air it plays only while the bed carries, and refused it says
// which key does work.
func TestSpacePlaysTheBedRelayAndNeverOverTheProgramme(t *testing.T) {
	plays := 0
	d, err := NewDashboard(Config{ToggleBedRelay: func() tea.Cmd { plays++; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = withStation(t, NewRouter(d), &station{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: consoleW, Height: consoleH}) // the refusal is read where the station row draws it (0.19.6 D-22)
	m, _ = m.Update(keyPress(t, "ctrl+b"))

	m, _ = m.Update(keyFor(t, "space"))
	if plays != 1 {
		t.Fatalf("space on standby played %d times; want the relay played", plays)
	}

	m, _ = m.Update(console.StationMsg{Power: lineup.Running})
	m, _ = m.Update(keyFor(t, "space"))
	if plays != 1 {
		t.Error("space on air played the relay over the programme (D-216)")
	}
	if got := drawn(m.(Router).broadcaster); !strings.Contains(got, bedRefusal) || !strings.Contains(bedRefusal, "[b]") {
		t.Errorf("the station row does not draw the refusal %q naming the cut key:\n%s", bedRefusal, got)
	}

	m, _ = m.Update(console.BedMsg{Carrying: true})
	if got := drawn(m.(Router).broadcaster); strings.Contains(got, bedRefusal) {
		t.Errorf("the bed carries and the refusal still reads %q", bedRefusal)
	}
	m, _ = m.Update(keyFor(t, "space"))
	if plays != 2 {
		t.Error("space while the bed carries did not play the relay")
	}

	// AND FROM OBSERVER SPACE IS OBSERVER'S: its own radio's key.
	r := m.(Router)
	r.active = SurfaceObserver
	_, _ = r.Update(keyFor(t, "space"))
	if plays != 2 {
		t.Error("space on Observer reached the console's bed")
	}
}

// STANDBY CLEARS THE PLAY KEY'S REFUSAL: off the air the sentence is false.
func TestStandbyClearsTheBedRefusal(t *testing.T) {
	d, err := NewDashboard(Config{ToggleBedRelay: func() tea.Cmd { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = withStation(t, NewRouter(d), &station{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: consoleW, Height: consoleH}) // the refusal is read where the station row draws it (0.19.6 D-22)
	m, _ = m.Update(keyPress(t, "ctrl+b"))
	m, _ = m.Update(console.StationMsg{Power: lineup.Running})
	m, _ = m.Update(keyFor(t, "space"))
	m, _ = m.Update(console.StationMsg{Power: lineup.OffAir})
	if got := drawn(m.(Router).broadcaster); strings.Contains(got, bedRefusal) {
		t.Errorf("on standby the refusal still reads %q", bedRefusal)
	}
}
