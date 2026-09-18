package tty

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/report"
)

// TestTheRunningOrderNamesWhatEachCardCarries.
//
// HUM LEAD, 2026-09-14: "only the Label in the Scheduled-Line up table in the
// base Broadcaster UI as comma delimited list: 02. NWS, FIRE, QUAKE" — and
// everything reads "Location Report, Full".
//
// ASKED OF THE SET, NOT SPELLED HERE. `report.Set.Describe` is the one owner of
// that rule, so the modal's summary line and this column cannot come to disagree
// about what a card carries — and a fifth report kind changes neither.
func TestTheRunningOrderNamesWhatEachCardCarries(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  report.Set
		want string
	}{
		{"a card that names nothing keeps the slot's name", 0, "Location Report"},
		{"everything", report.Everything(), report.FullLabel},
		{"three of four, in registry order",
			report.Set(0).Add(report.NWS).Add(report.Fire).Add(report.Seismic), "NWS, FIRE, QUAKE"},
		{"one is a one-element list", report.Set(0).Add(report.Marine), "MARINE"},
	} {
		c := card(t, "c0", "somewhere")
		c.Reports = tc.set
		if got := reportTypeOf(c); got != tc.want {
			t.Errorf("%s: the running order says %q, want %q", tc.name, got, tc.want)
		}
	}
}
