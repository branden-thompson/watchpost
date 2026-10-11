package render

import (
	"testing"
	"time"
)

// TestTheFreshnessRampHasSevenRungs is 0.19.0 D-166: the general-use ramp
// for a datum's age - just now, then under 5, 15, 30, 45 and 60 minutes,
// then an hour or more - each rung its own token, each at its edge.
func TestTheFreshnessRampHasSevenRungs(t *testing.T) {
	for _, c := range []struct {
		age  time.Duration
		want Token
	}{{0, FreshNow}, {59 * time.Second, FreshNow}, {time.Minute, Fresh5}, {5*time.Minute - time.Second, Fresh5},
		{5 * time.Minute, Fresh15}, {15*time.Minute - time.Second, Fresh15}, {15 * time.Minute, Fresh30}, {30*time.Minute - time.Second, Fresh30},
		{30 * time.Minute, Fresh45}, {45*time.Minute - time.Second, Fresh45}, {45 * time.Minute, Fresh60}, {time.Hour - time.Second, Fresh60},
		{time.Hour, FreshOld}, {48 * time.Hour, FreshOld}, {-time.Minute, FreshNow}} {
		if got := FreshnessTone(c.age); got != c.want {
			t.Errorf("%v: %s; want %s", c.age, got, c.want)
		}
	}
}

// TestTheFreshnessRampTakesTheThemesHues is D-166: the default theme's ramp
// is its own light blue, green, yellow, orange and red where it has them,
// light green and green-yellow (#999000) between; every rung a distinct
// colour; a theme with no colour gives the ramp none.
func TestTheFreshnessRampTakesTheThemesHues(t *testing.T) {
	d := defaultTheme()
	seen := map[string]Token{}
	for _, tok := range []Token{FreshNow, Fresh5, Fresh15, Fresh30, Fresh45, Fresh60, FreshOld} {
		v := d[tok]
		if v == "" {
			t.Fatalf("%s has no colour in the default theme", tok)
		}
		if other, dup := seen[v]; dup {
			t.Errorf("%s and %s are the same colour, %s", tok, other, v)
		}
		seen[v] = tok
	}
	own := withFreshLadder(map[Token]string{FocusCell: "blue", ProviderOK: "green", AlertLabel: "yellow", FireMark: "orange", AlertDanger: "red", Fresh45: "named"},
		map[Token]string{Fresh45: "named"}) // a rung the theme names is its own
	for tok, want := range map[Token]string{FreshNow: "blue", Fresh5: "green", Fresh15: freshLightGreen, Fresh30: freshGreenYellow, Fresh45: "named", Fresh60: "orange", FreshOld: "red"} {
		if own[tok] != want {
			t.Errorf("%s is %q; want %q", tok, own[tok], want)
		}
	}
	if freshGreenYellow != "38;2;153;144;0" {
		t.Errorf("the green-yellow is %s; the HUM LEAD named #999000", freshGreenYellow)
	}
}
