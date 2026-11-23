package tty

// extended_days_test.go — the day cells are the caller's ask: Observer's row
// carries them and the console's asks for none (D-120).

import (
	"fmt"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/internal/frame"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// AND OBSERVER STILL GETS ITS EXTENDED DAYS (D-120).
//
// THE CONSOLE SKIPS THEM and Observer draws them, which is the whole shape of
// the rule: the day cells are the CALLER's ask. A skip that reached Observer
// would empty its ultra-wide columns — a regression in the shipped surface, paid
// for by an optimisation on the new one, which is the trade the release posture
// exists to refuse.
func TestObserverKeepsItsExtendedDays(t *testing.T) {
	hi, lo := 31.0, 12.0
	daily := make([]snapshot.Daily, 0, 7)
	for d := range 7 { // bounded by the forecast (P10-02)
		daily = append(daily, snapshot.Daily{Date: fmt.Sprintf("2026-09-%02d", 13+d),
			Condition: "CLEAR", TempMax: &hi, TempMin: &lo})
	}
	loc := &snapshot.Location{Label: "Oceanside, CA", Zip: "92057", Daily: daily}

	if got := frame.WeatherRow(loc, 50, extRowDays); len(got.Extended) != 5 {
		t.Errorf("Observer's row carries %d extended days, want 5", len(got.Extended))
	}
	// AND THE CONSOLE ASKS FOR NONE, which is the saving.
	if got := frame.WeatherRow(loc, 50, 0); len(got.Extended) != 0 {
		t.Errorf("the console's row built %d extended days it never draws", len(got.Extended))
	}
}
