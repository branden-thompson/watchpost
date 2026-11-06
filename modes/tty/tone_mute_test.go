package tty

import (
	"maps"
	"slices"
	"testing"
)

// The tests in this file are 0.19.5 Q2.2 (RN-2): muting an alert-tone class in
// Settings, whose rule mirrors cast.Muted: under mute mode an empty set means
// every class.

// toneRows is each class row's id and key, in the rows' order.
func toneRows(t *testing.T) ([]setupRowID, []string) {
	t.Helper()
	ids := classRowOrder()
	if len(ids) < 2 {
		t.Fatalf("Settings has %d tone rows; the test needs at least two", len(ids))
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = classKeyOf(id)
	}
	return ids, keys
}

// TestEnablingOneClassUnderAnEmptyMuteKeepsTheOthersMuted: "Mute" with nothing
// ticked mutes every class, so enabling one writes the others down first, or
// every class would stay muted, the one just enabled among them.
func TestEnablingOneClassUnderAnEmptyMuteKeepsTheOthersMuted(t *testing.T) {
	ids, keys := toneRows(t)
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	d.setup.toneMode, d.setup.toneMuted = toneModeMute, nil
	for _, id := range ids {
		if w := d.toneStateWord(id); w != "MUTED" {
			t.Fatalf("under mute with nothing ticked, %s says %q; want MUTED", classKeyOf(id), w)
		}
	}
	d = d.toggleClass(ids[0])
	if d.setup.toneMode != toneModeMute {
		t.Errorf("enabling one class turned mute mode off; the other classes are still muted")
	}
	want := keys[1:]
	if got := slices.Sorted(maps.Keys(d.setup.toneMuted)); !slices.Equal(got, slices.Sorted(slices.Values(want))) {
		t.Errorf("the muted set is %v; want every class but %s: %v", got, keys[0], want)
	}
	if w := d.toneStateWord(ids[0]); w != "Enabled" {
		t.Errorf("the class just enabled says %q", w)
	}
	if w := d.toneStateWord(ids[1]); w != "MUTED" {
		t.Errorf("a class left muted says %q", w)
	}
}

// TestEnablingTheLastMutedClassTurnsMuteModeOff: no row may say MUTED because of
// an empty set nobody asked for.
func TestEnablingTheLastMutedClassTurnsMuteModeOff(t *testing.T) {
	ids, keys := toneRows(t)
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	d.setup.toneMode, d.setup.toneMuted = toneModeMute, map[string]bool{keys[0]: true}
	d = d.toggleClass(ids[0])
	if d.setup.toneMode != toneModeOn || len(d.setup.toneMuted) != 0 {
		t.Errorf("after the last muted class was enabled: mode %q, set %v; want all tones on", d.setup.toneMode, d.setup.toneMuted)
	}
	for _, id := range ids {
		if w := d.toneStateWord(id); w != "Enabled" {
			t.Errorf("%s says %q with every tone on", classKeyOf(id), w)
		}
	}
}
