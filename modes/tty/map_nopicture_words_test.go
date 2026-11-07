package tty

// map_nopicture_words_test.go — F-205 AX-5 (FR-7.4, A-107 (7)): without the
// picture, what the picture's rows say is said as text - the status, the
// timeline, and the region beyond an edge.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestWithoutThePictureTheStatusAndTimelineAreText is AX-5: in place of the
// picture, the loop's line - its source, frame and state - is in the text,
// and a press of play changes it at once; a view still loading says so.
func TestWithoutThePictureTheStatusAndTimelineAreText(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	d.mapDesc = mapDescInstead
	text := bodyText(d)
	for _, want := range []string{"RADAR", "FRAME 12 / 12", "STOPPED"} {
		if !strings.Contains(text, want) {
			t.Errorf("without the picture the text does not say %q:\n%s", want, text)
		}
	}
	m, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if got := bodyText(m); !strings.Contains(got, "PLAYING") {
		t.Errorf("play is not said at once without the picture:\n%s", got)
	}
	loading := mapDash(t, Config{NewMap: offlineMap, MapDescription: "instead"})
	loading, _ = pressKey(loading, "g")
	if got := bodyText(loading); !strings.Contains(got, mapLoadingText) {
		t.Errorf("a view still loading is not said without the picture:\n%s", got)
	}
}

// TestWithoutThePictureTheEdgeIsSaid is AX-5 (D-81): at a region's edge the
// text says the region beyond and that the next press crosses, in words a
// terminal without the picture's arrows can print.
func TestWithoutThePictureTheEdgeIsSaid(t *testing.T) {
	d := openMap(t, Config{MapDescription: "off"}, 133, 44)
	d = untilChip(t, pressCode(d, '1', "1"), tea.KeyRight)
	d.mapDesc = mapDescInstead
	if got := bodyText(d); !strings.Contains(got, "At the east edge: press again to cross to US Caribbean.") {
		t.Errorf("without the picture the edge is not said:\n%s", got)
	}
}
