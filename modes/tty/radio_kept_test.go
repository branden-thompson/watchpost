package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// THE RADIO PANEL'S CHOICES ARE KEPT (D-214): the volume - the console's gain
// with it - the repeat and the visualizer open as they were left, and each
// press that changes one saves the three.
func TestTheRadioPanelsChoicesAreKept(t *testing.T) {
	var saved []RadioPrefs
	fr := &fakeRadio{}
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	d = d.WithRadio(fr).WithRadioPrefs(RadioPrefs{Volume: 30, Repeat: RepeatOne, Viz: true}, func(p RadioPrefs) error {
		saved = append(saved, p)
		return nil
	})
	if d.radioVolume != 30 || d.radioRepeat != RepeatOne || !d.radioViz {
		t.Fatalf("the panel opens at %d, %v, viz %v; want what was kept", d.radioVolume, d.radioRepeat, d.radioViz)
	}
	press := func(d Dashboard, r rune) Dashboard {
		m, cmd := d.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		for _, msg := range msgsOf(t, cmd) {
			m, _ = m.(Dashboard).Update(msg)
		}
		return m.(Dashboard)
	}
	d = press(d, '+')
	d = press(d, 'r')
	d = press(d, 'v')
	if len(saved) != 3 {
		t.Fatalf("three changes saved %d times; want each saved", len(saved))
	}
	if last := saved[2]; last.Volume != 35 || last.Repeat != RepeatWatchlist || last.Viz {
		t.Errorf("the last save is %+v; want 35, Watchlist, visualizer off", last)
	}
	if fr.vol != 35 || fr.repeat != RepeatWatchlist {
		t.Errorf("the radio is at %d, %v; the saves must not take the place of telling it", fr.vol, fr.repeat)
	}
}

// A KEPT REPEAT REACHES THE RADIO AT LAUNCH: the panel opening on Repeat One
// tells the player so, or the chip says One while the player stops at the end.
func TestAKeptRepeatReachesTheRadioAtLaunch(t *testing.T) {
	fr := &fakeRadio{}
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	d = d.WithRadio(fr).WithRadioPrefs(RadioPrefs{Volume: 55, Repeat: RepeatOne}, nil)
	for _, msg := range msgsOf(t, d.Init()) {
		_, _ = d.Update(msg)
	}
	if fr.repeat != RepeatOne {
		t.Errorf("the radio was told %v at launch; want the kept Repeat One", fr.repeat)
	}
}
