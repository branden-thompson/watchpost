package render

import (
	"strings"
	"testing"
)

// The width ladder (NFR-11): EXPIRES drops first, then DECLARED, then
// LOCATION squeezes to 16; EVENT keeps ≥ 22 until nothing is left to drop.
// For every inner width from the 80-col floor (66) to the 110-col ceiling
// (103) the columns sum to exactly inner; below 44 the ladder bottoms out
// (EVENT 14 + LOCATION 16 + chrome) and the lines run over — outside the
// spec, pinned here so a change is deliberate (R3-B-10).
func TestSevereColumnsLadder(t *testing.T) {
	cases := []struct {
		inner int
		ev    int
		cols  []string
	}{
		{119, 24, []string{"LOCATION", "DETECTION", "DECLARED", "EXPIRES"}}, // 133 cols: every column
		{106, 30, []string{"LOCATION", "DETECTION", "DECLARED"}},            // 120 cols: EXPIRES dropped
		{86, 31, []string{"LOCATION", "DECLARED"}},                          // 100 cols: DETECTION dropped
		{66, 30, []string{"LOCATION"}},                                      // 80 cols: DECLARED dropped
		{50, 20, []string{"LOCATION"}},                                      // below the floor: LOCATION squeezed to 16
		{40, 14, []string{"LOCATION"}},                                      // the ladder's bottom
	}
	for _, c := range cases {
		ev, cols := severeColumns(c.inner)
		names := make([]string, len(cols))
		sum := severeMarksW + severeNumW + ev
		for i, col := range cols {
			names[i] = col.name
			sum += severeGutter + col.width
		}
		if ev != c.ev || strings.Join(names, ",") != strings.Join(c.cols, ",") {
			t.Errorf("inner %d: EVENT %d cols %v, want %d %v", c.inner, ev, names, c.ev, c.cols)
		}
		if c.inner >= 44 && sum != c.inner {
			t.Errorf("inner %d: columns sum to %d", c.inner, sum)
		}
	}
	for inner := 66; inner <= 123; inner++ {
		lines := Opts{Width: inner}.SevereTable([]SevereCell{{Num: 1, Event: strings.Repeat("x", 80), Location: strings.Repeat("y", 40), Declared: "08/28 11:20 EDT", Expires: "08/28 20:00 EDT", Focused: true}}, inner, EventCatOrangeBG)
		for i, l := range lines {
			if w := Width(l); w != inner {
				t.Fatalf("inner %d line %d is %d wide: %q", inner, i, w, l)
			}
		}
	}
}

func TestSevereMarksAreFiveCells(t *testing.T) {
	o := Opts{}
	for _, c := range []SevereCell{{Focused: true}, {Playing: true}, {Focused: true, Playing: true}, {}} {
		if w := Width(severeMarks(o, c)); w != severeMarksW {
			t.Errorf("%+v: %d cells", c, w)
		}
	}
	if got := severeMarks(Opts{ASCII: true}, SevereCell{Focused: true}); got != ">    " { // the pointer alone: ▶ only while that event plays (UAT item 11)
		t.Errorf("ascii marks focused: %q", got)
	}
	if got := severeMarks(Opts{ASCII: true}, SevereCell{Focused: true, Playing: true}); got != ">  * " { // the play mark is its own ASCII form (B-08b ruling)
		t.Errorf("ascii marks playing: %q", got)
	}
}

// The headers are the dashboard's group bands: bracketed spaced lettering
// with colour off, GroupText on the header band with colour on.
func TestSevereHeadersAreGroupBands(t *testing.T) {
	if got := severeHeader("EVENT", 15, true, ""); got != "[  E V E N T  ]" {
		t.Errorf("spread: %q", got)
	}
	if got := severeHeader("DECLARED", 15, false, ""); got != "[  DECLARED   ]" {
		t.Errorf("plain: %q", got)
	}
	ev, data := severeColumns(119)
	line := severeHeaderLine(ev, data, "", false)
	if Width(line) != 119 || !strings.HasPrefix(line, "[") || strings.Index(line, "]") != severeMarksW+severeNumW+ev+severeGutter/2-1 { // + the half gutter it takes
		t.Errorf("EVENT must span marks+num+event and half the gutter: %q", line)
	}
	if strings.Count(line, "][") != len(data) { // the bands touch in the gutters (item 10)
		t.Errorf("header bands must meet: %q", line)
	}
	for _, want := range []string{"LOCATION", "DETECTION", "DECLARED", "EXPIRES"} {
		if !strings.Contains(line, " "+want+" ") || strings.Contains(line, strings.Join(strings.Split(want, ""), " ")) {
			t.Errorf("%s must read plain: %q", want, line)
		}
	}
	rows := Opts{Width: 119}.SevereTable([]SevereCell{{Num: 1, Event: "x", Location: "y", Detection: "Observed", Declared: "d", Expires: "e"}}, 119, EventCatOrangeBG)
	if !strings.Contains(rows[1], "x"+strings.Repeat(" ", ev-1)+strings.Repeat(" ", severeGutter)+"y") { // the row keeps its gutter (item 10b)
		t.Errorf("rows keep their gutters: %q", rows[1])
	}
}

// Railify's thumb tracks the scroll position over ALL the lines given: the
// first line at the top, the last at the bottom (R3-B-05: a caller that
// draws ▼ on its last row passes the rows above it).
func TestRailifyThumbTracksTheScroll(t *testing.T) {
	g := RailGlyphsFor(false)
	rows := []string{"a", "b", "c", "d"}
	top := Railify(rows, 6, 0, 10, 4, g)
	bottom := Railify(rows, 6, 6, 10, 4, g)
	if !strings.HasSuffix(top[0], g.Thumb) || strings.Count(strings.Join(top, ""), g.Thumb) != 1 {
		t.Errorf("top: %q", top)
	}
	if !strings.HasSuffix(bottom[3], g.Thumb) {
		t.Errorf("bottom: %q", bottom)
	}
	if a := RailGlyphsFor(true); a != (RailGlyphs{"^", "v", "#", "|"}) {
		t.Errorf("ascii rail: %+v", a)
	}
}
