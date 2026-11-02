package tty

// map_layer_words_test.go — F-205 AX-4 (FR-7.4, D-363): the description
// says, after the alerts, what each layer drawn shows at the place.

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// TestEachLayerDrawnAddsItsSentence is D-363: one sentence a layer, after the
// alerts, from the answer for what the moment shown draws - Forecast mode's
// other days are held but not said; a radar loop says its class and the
// nearest heavier echo; each shape of answer has its words.
func TestEachLayerDrawnAddsItsSentence(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	words := d.describeLinesAll()
	temp := slices.IndexFunc(words, func(l string) bool { return strings.HasPrefix(l, "Temperature at Oceanside, CA: ") })
	alert := slices.IndexFunc(words, func(l string) bool { return strings.HasPrefix(l, "Wind Warning") })
	if temp < 0 || alert < 0 || temp < alert {
		t.Fatalf("the temperature is not said after the alerts:\n%s", strings.Join(words, "\n"))
	}
	if n := strings.Count(strings.Join(words, "\n"), "Temperature at"); n != 1 {
		t.Errorf("the temperature is said %d times; once, for the moment shown", n)
	}
	now := words[temp]
	m, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	if next := m.(Dashboard).describeLinesAll(); slices.Contains(next, now) {
		t.Errorf("a step on, the temperature still says Now's: %q", now)
	}
	var asked []string
	r := openRadarMap(t, "MRMS", &asked)
	if got := strings.Join(r.describeLinesAll(), "\n"); !strings.Contains(got, "Radar: no echo at Oceanside, CA; the nearest heavier echo is 18 miles to the south-west.") {
		t.Errorf("the radar's answer is not said:\n%s", got)
	}
	for _, c := range []struct {
		a    tuimaps.Answer
		want string
	}{
		{tuimaps.Answer{Form: answerField, Value: 12.4, ValueUnit: "mph", From: "south-west"}, "Wind at Oceanside, CA: 12 mph from the south-west."},
		{tuimaps.Answer{Form: answerField, NoData: true}, "Wind: no reading at Oceanside, CA."},
		{tuimaps.Answer{Form: "points", Label: "M3.1", Distance: 22.6, Unit: "miles", Compass: "north"}, "Wind: the nearest, M3.1, is 23 miles to the north."},
		{tuimaps.Answer{Form: answerArea, Relation: tuimaps.Inside}, "Oceanside, CA lies inside the wind."},
		{tuimaps.Answer{Form: answerArea, Distance: 0.4, Unit: "miles", Compass: "east"}, "Wind: the nearest edge is under one mile to the east."},
	} {
		if got := r.answerSentence("Wind", "Oceanside, CA", c.a); got != c.want {
			t.Errorf("%+v says %q; want %q", c.a, got, c.want)
		}
	}
}

// TestTheRadarSaysItsMotion is D-363: while the heavier rain moves, the
// radar's sentence is followed by where it is, which way it goes and how
// fast; held still, nothing more is said.
func TestTheRadarSaysItsMotion(t *testing.T) {
	var asked []string
	r := openRadarMap(t, "MRMS", &asked)
	r.mapPane.motion = []tuimaps.MotionReport{{Overlay: "radar/us-a", Moving: true, HeadingCompass: "east", SpeedKmh: 32.2, Trend: tuimaps.Closer,
		To: tuimaps.Sighting{Distance: 18, Unit: "miles", Compass: "south-west"}}}
	words := r.describeLinesAll()
	radar := slices.IndexFunc(words, func(l string) bool { return strings.HasPrefix(l, "Radar: ") })
	want := "Heavier rain 18 miles to the south-west of Oceanside, CA moves towards the east at 20 miles an hour, coming closer."
	if radar < 0 || radar+1 >= len(words) || words[radar+1] != want {
		t.Errorf("the motion does not follow the radar's sentence:\n%s", strings.Join(words, "\n"))
	}
	r.mapPane.motion[0].Moving = false
	if strings.Contains(strings.Join(r.describeLinesAll(), "\n"), "Heavier rain") {
		t.Error("rain that barely moved is said to move")
	}
}
