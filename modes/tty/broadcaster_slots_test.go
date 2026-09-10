package tty

// broadcaster_slots_test.go — the console draws its ten slots ALWAYS (D-64).
//
// HUM LEAD, UAT 2026-09-10: "Broadcaster UI should adopt the same methodology
// as Observer — render the full Broadcaster Dashboard, including all 10 cards …
// Cards should SHIMMER while the producers are proposing and the director is
// deciding … Right now this looks broken, so if I was a user who came upon this,
// I would not expect this to be working as is."
//
// "(nothing scheduled)" IS A DEAD END. It says the station has nothing and gives
// the operator nowhere to look; ten shimmering slots say the machine is working
// and the data is coming. Observer has said that since UAT 18.2 — this is the
// console adopting it rather than inventing a second answer.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

func TestTheConsoleAlwaysDrawsItsTenSlots(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")

	boxes := 0
	for _, r := range rows {
		if strings.Contains(r, "+---") {
			boxes++
		}
	}
	// Each card is a box: a top border and a bottom one.
	if got := boxes / 2; got != MainTrackSlots {
		t.Errorf("an empty line-up still draws its %d slots; got %d", MainTrackSlots, got)
	}
	if strings.Contains(stripANSITest(b.View().Content), "nothing scheduled") {
		t.Error(`"(nothing scheduled)" is a dead end; the slots say it better`)
	}
}

// AN EMPTY SLOT SHIMMERS, and it is the SAME shimmer Observer uses — one owner
// for "this is loading", so the two surfaces cannot come to mean different
// things by it.
func TestAnEmptySlotShimmers(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	first := stripANSITest(b.View().Content)
	b.frame++
	second := stripANSITest(b.View().Content)

	if first == second {
		t.Error("an empty slot must animate between frames; the console is static")
	}
}

// A FILLED SLOT DOES NOT SHIMMER. The animation means "still deciding", so a
// card the Director has already chosen must stop it.
func TestAFilledSlotIsStill(t *testing.T) {
	var l lineup.Lineup
	c := card(t, "a", "LOCATION REPORT • OCEANSIDE, CA")
	next, err := l.Queue(lineup.MainTrack, c)
	if err != nil {
		t.Fatalf("seeding: %v", err)
	}
	l = next
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l})

	rows := strings.Split(stripANSITest(b.View().Content), "\n")
	for _, r := range rows {
		if strings.Contains(r, "OCEANSIDE") && strings.Contains(r, "...") {
			t.Errorf("a decided card must not shimmer:\n%q", r)
		}
	}
}

// AND THE SLOT NUMBERS ARE STILL THE ADDRESS. An empty slot is addressable —
// the operator can put something in it — so it carries its handle.
func TestAnEmptySlotStillCarriesItsHandle(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	got := stripANSITest(b.View().Content)
	for _, want := range []string{chipFor("0"), chipFor("9")} {
		if !strings.Contains(got, want) {
			t.Errorf("every slot is addressable; %q is missing", want)
		}
	}
}

// THE SHIMMER IS DRIVEN BY THE TICK, AND THAT IS THE PART A DIRECT `b.frame++`
// NEVER TESTS.
//
// The shimmer test above advances the phase by hand, so it proves the DRAWING
// animates and says nothing about whether anything ever advances it. Three
// plants survived on exactly that gap — the frame never incrementing, the tick
// never arming, and the tick never stopping.
func TestTheTickAdvancesTheShimmerAndReArms(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true

	before := b.frame
	b, cmd := b.Update(tickMsg{})

	if b.frame == before {
		t.Errorf("the tick must advance the shimmer's phase; it stayed at %d", before)
	}
	if cmd == nil {
		t.Error("and it must re-arm while slots are still waiting, or the shimmer runs once and stops")
	}
}

// AND IT STOPS WHEN THERE IS NOTHING LEFT TO ANIMATE. A tick that kept firing
// would redraw a console nobody is watching change — the same discipline
// Observer's own `tickNeeded` applies.
func TestTheTickStopsOnceEverySlotIsDecided(t *testing.T) {
	var l lineup.Lineup
	for i := 0; i < MainTrackSlots; i++ {
		c := card(t, "c"+string(rune('a'+i)), "LOCATION REPORT • TOWN "+string(rune('A'+i)))
		next, err := l.Queue(lineup.MainTrack, c)
		if err != nil {
			t.Fatalf("seeding: %v", err)
		}
		l = next
	}
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l})

	if b.tickNeeded() {
		t.Fatal("a full line-up has nothing left to animate")
	}
	if _, cmd := b.Update(tickMsg{}); cmd != nil {
		t.Error("a full line-up must not re-arm the shimmer")
	}
}
