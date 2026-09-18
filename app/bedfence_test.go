package app

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/platform/config"
)

// THE OPERATOR IS TOLD WHAT THEIR FENCE GETS THEM (D-77).
//
//	"we should provide the user feedback on how many relays they get based on
//	 their radius … and if the user increases / decreases we call out the various
//	 risks associated with it" — HUM LEAD, 2026-09-10
//
// THE THRESHOLDS ARE MEASURED. Around Bonsall a 25-mile fence reaches ONE
// transmitter and 100 reaches eight; Lone Pine reaches NONE inside fifty miles.
func TestTheOperatorIsToldWhatTheBedFenceReaches(t *testing.T) {
	table, err := stream.LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	const lat, lon = 33.2881, -117.2256 // Bonsall

	// THE RULED DEFAULT IS A REAL CHOICE, AND IT CARRIES NO WARNING. A default
	// that arrives wearing one is a warning the operator learns to ignore.
	def := bedReachFor(table, lat, lon, config.DefaultBedRadiusMi)
	if def.Relays < 6 {
		t.Errorf("the ruled default reaches %d relays; it was chosen because it reaches several", def.Relays)
	}
	if def.Advice != "" {
		t.Errorf("the default carries a warning: %q", def.Advice)
	}

	// ONE RELAY IS NOT A SELECTOR, and the operator is told so.
	if got := bedReachFor(table, lat, lon, 25); got.Relays != 1 || !strings.Contains(got.Advice, "nothing to switch between") {
		t.Errorf("a 25-mile fence: %d relays, advice %q", got.Relays, got.Advice)
	}

	// AND LONE PINE — the HUM LEAD's own counter-example — reaches none at all.
	if got := bedReachFor(table, 36.6060, -118.0640, 50); got.Relays != 0 || !strings.Contains(got.Advice, "nothing to carry") {
		t.Errorf("Lone Pine at 50 miles: %d relays, advice %q", got.Relays, got.Advice)
	}
}

// THE RISK AT THE OTHER END IS THE OBJECTION THAT STARTED ALL OF THIS: a relay
// far enough out is broadcasting a forecast for a region the station's listeners
// are not in.
func TestAWideBedFenceIsCalledOutToo(t *testing.T) {
	if got := bedAdvice(20, config.MaxBedRadiusMi); !strings.Contains(got, "different forecast area") {
		t.Errorf("a 150-mile fence says nothing about reach: %q", got)
	}
	// AND AN EMPTY FENCE IS WARNED ABOUT BEFORE A WIDE ONE IS: a station with no
	// bed has a problem, not a trade-off.
	if got := bedAdvice(0, config.MaxBedRadiusMi); !strings.Contains(got, "nothing to carry") {
		t.Errorf("a wide fence with no relays warns about the wrong thing: %q", got)
	}
}

// THE LINE IS ONE OWNER'S, so Settings and the console cannot say different
// things about one fence.
func TestTheReachReadsAsASentence(t *testing.T) {
	if got := (bedReach{Relays: 1, Advice: "x"}).Line(); got != "1 relay in range — x" {
		t.Errorf("one relay is singular: %q", got)
	}
	if got := (bedReach{Relays: 8}).Line(); got != "8 relays in range" {
		t.Errorf("no advice is no dash: %q", got)
	}
	if got := (bedReach{}).Line(); got != "0 relays in range" {
		t.Errorf("zero is plural: %q", got)
	}
}
