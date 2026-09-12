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

// THE CONSOLE DRAWS EVERY SLOT IT HAS ROOM FOR, AND NEVER A DEAD END (D-64).
//
// IT USED TO DRAW ALL TEN AT ONCE, and D-87 made the cards taller than that: a
// card is a manifest now, so the ten slots need about ninety rows and the
// terminal the reference is drawn at has seventy-four. The queue scrolls, which
// is what its rail has always been for.
//
// WHAT DID NOT CHANGE is the rule D-64 exists for: a station with nothing
// scheduled shows SLOTS rather than a sentence saying it has nothing.
func TestTheConsoleDrawsEverySlotItHasRoomFor(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")

	boxes := 0
	for _, r := range rows {
		if strings.Contains(r, "+---") {
			boxes++
		}
	}
	// TWO BOXES NOW, NOT TEN (D-94). Slots 0 and 1 are the cards the operator
	// reads from and keep their boxes; everything below is a TABLE, so counting
	// boxes counts the read region and nothing else.
	if got := boxes / 2; got != 2 {
		t.Errorf("an empty line-up drew %d boxed slots; LIVE and UP NEXT are the two that keep a box", got)
	}
	// AND THE TABLE STILL DRAWS ITS SLOTS, which is where the rest of the running
	// order went.
	for _, want := range []string{"02.", "03."} {
		if !strings.Contains(stripANSITest(b.View().Content), want) {
			t.Errorf("an empty line-up drew no %q row; a slot is an ADDRESS whether or not it is filled", want)
		}
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
//
// EXCEPT SLOT 0 ON A STATION AT REST (D-89), which is the one slot the operator
// CANNOT put anything in: the line-up is drawn from UP NEXT down while nothing is
// on the air (liveOffset), and `SHIFT+ENTER` is what fills slot 0 rather than any
// per-slot control. It draws the standby box, and that box carries no handle
// because `[0]` would open nothing — which is the defect F-97 was filed for.
func TestAnEmptySlotStillCarriesItsHandle(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	got := stripANSITest(b.View().Content)
	// THE READ CARD KEEPS ITS CHIP; THE TABLE ROWS CARRY THEIR NUMBER (D-94).
	// The `##.` column IS the address, exactly as it is on Observer's table, so a
	// chip beside it would be the address written twice.
	if !strings.Contains(got, chipFor("1")) {
		t.Errorf("the UP NEXT card is not addressable; %q is missing", chipFor("1"))
	}
	for _, want := range []string{"02.", "04."} {
		if !strings.Contains(got, want) {
			t.Errorf("every slot is addressable; the %q row is missing", want)
		}
	}
	if strings.Contains(got, chipFor("0")) {
		t.Errorf("the standby box carries %q, which addresses nothing", chipFor("0"))
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
