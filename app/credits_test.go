package app

import (
	"strings"
	"testing"
)

func TestCreditsCoverEverySourceAndFitTheAboutWindow(t *testing.T) {
	// UAT 75 (OQ-15): every data source the build reads is credited —
	// GeoNames and Open-Meteo are CC BY 4.0, so the two CC lines are a
	// licence obligation — and every line fits the About window's 52-cell
	// text width (60 cols, 3-cell inset each side).
	lines := credits()
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"National Weather Service", "Data Buoy Center", "Tides & Currents", "GeoNames", "CC BY 4.0", "Open-Meteo", "NWR transmitter", "wxradio.org", "not for life-safety"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("credits missing %q:\n%s", want, joined)
		}
	}
	if strings.Count(joined, "CC BY 4.0") != 2 {
		t.Fatalf("both CC BY sources must carry their licence:\n%s", joined)
	}
	for _, l := range lines {
		if n := len([]rune(l)); n > 52 {
			t.Fatalf("credit line exceeds the About window's 52 cells (%d): %q", n, l)
		}
	}
}
