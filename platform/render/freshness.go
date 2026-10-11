package render

import "time"

// The freshness ramp (0.19.0 D-166): a general-use colour for how old a
// datum is, seven rungs. It borrows the theme's own light blue, green,
// yellow, orange and red where the theme has them, with a light green and a
// green-yellow (#999000) between; withFreshLadder fills it for every theme.
// The Broadcaster's data-pull ladder (DataNew to DataStale) is its own, by
// its own ruling, and is not this.
const (
	FreshNow Token = "fresh.now" // just now, under a minute: light blue
	Fresh5   Token = "fresh.5"   // under 5 minutes: green
	Fresh15  Token = "fresh.15"  // under 15: light green
	Fresh30  Token = "fresh.30"  // under 30: green-yellow, #999000
	Fresh45  Token = "fresh.45"  // under 45: yellow
	Fresh60  Token = "fresh.60"  // under 60: orange
	FreshOld Token = "fresh.old" // an hour or more: red
)

// FreshnessTone is the ramp's rung for an age; an age before now (a clock
// ahead of the source's) is just now.
func FreshnessTone(age time.Duration) Token {
	switch {
	case age < time.Minute:
		return FreshNow
	case age < 5*time.Minute:
		return Fresh5
	case age < 15*time.Minute:
		return Fresh15
	case age < 30*time.Minute:
		return Fresh30
	case age < 45*time.Minute:
		return Fresh45
	case age < time.Hour:
		return Fresh60
	}
	return FreshOld
}

// The ramp's two rungs no theme colour names, as the default gives them: a
// light green (#86EFAC) and the HUM LEAD's green-yellow, #999000.
const (
	freshLightGreen  = "38;2;134;239;172"
	freshGreenYellow = "38;2;153;144;0"
)

// withFreshLadder fills the freshness ramp from the theme's own colours
// where the theme has not named a rung, as withAgeLadder fills its ladder.
func withFreshLadder(t map[Token]string, named map[Token]string) map[Token]string {
	for _, f := range []struct{ dst, src Token }{
		{FreshNow, FocusCell},   // the theme's own light blue
		{Fresh5, ProviderOK},    // the green it calls healthy
		{Fresh45, AlertLabel},   // the yellow it calls caution
		{Fresh60, FireMark},     // the orange it calls hot
		{FreshOld, AlertDanger}, // the red it calls warning
	} {
		if _, ok := named[f.dst]; !ok {
			t[f.dst] = t[f.src]
		}
	}
	for tok, v := range map[Token]string{Fresh15: freshLightGreen, Fresh30: freshGreenYellow} {
		if _, ok := named[tok]; !ok && t[tok] == "" {
			t[tok] = v
		}
	}
	return t
}
