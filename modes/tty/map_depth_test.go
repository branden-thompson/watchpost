package tty

// map_depth_test.go — 0.19.0 D-207: the map's colour depth, a Map Setting.

import (
	"strings"
	"testing"
)

// TestTheColourDepthIsASetting is D-207: Automatic by default, the map
// drawn at the depth the terminal says; a depth chosen is the map's at its
// next frame, and Automatic chosen again hands it back to the terminal;
// with NO_COLOR set, none, whatever is chosen; the choice is the file's word.
func TestTheColourDepthIsASetting(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")
	d := openMap(t, Config{}, 133, 44)
	frame := func(d Dashboard) string { return strings.Join(d.renderMap().mapPane.lines, "\n") }
	truecolor, palette := "38;2;", "38;5;"
	if mapDepthLabel(d.mapDepth) != "Automatic" || strings.Contains(frame(d), truecolor) || !strings.Contains(frame(d), palette) {
		t.Fatalf("by default the Setting is %q and the map not the terminal's 256 colours", mapDepthLabel(d.mapDepth))
	}
	d = d.cycleMapDepth(true)
	if mapDepthLabel(d.mapDepth) != "Truecolor" || !strings.Contains(frame(d), truecolor) {
		t.Errorf("Truecolor chosen (%q), the map is not drawn truecolor", mapDepthLabel(d.mapDepth))
	}
	for range len(mapDepthChoices) - 1 {
		d = d.cycleMapDepth(true)
	}
	if mapDepthLabel(d.mapDepth) != "Automatic" || strings.Contains(frame(d), truecolor) {
		t.Errorf("Automatic chosen again (%q), the map keeps truecolor", mapDepthLabel(d.mapDepth))
	}
	d.mapDepth, d.cfg.NoColor = depthTruecolor, true
	t.Setenv("NO_COLOR", "1")
	if f := frame(d); strings.Contains(f, truecolor) || strings.Contains(f, palette) {
		t.Error("with NO_COLOR set the map draws colour")
	}
	for c, key := range map[mapDepth]string{depthAuto: "", depthTruecolor: "truecolor", depth256: "256", depth16: "16", depthNone: "none"} {
		if mapDepthKey(c) != key || mapDepthByKey(key) != c {
			t.Errorf("%s is the file's %q", mapDepthLabel(c), mapDepthKey(c))
		}
	}
	if mapDepthByKey("bogus") != depthAuto {
		t.Error("a word not offered is not Automatic")
	}
}
