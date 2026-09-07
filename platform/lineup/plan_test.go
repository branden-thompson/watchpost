package lineup

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
)

var updatePlan = flag.Bool("update-plan", false, "re-capture testdata/plan.golden")

// planNow is the fixed clock every test below plans against. No assertion path
// reads the real clock (DR-20) — the freshness rule is the whole reason the
// clock is injected, and a plan that could not be replayed exactly is not a
// plan anyone can reason about.
var planNow = time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

// arrival is one alert offered to the burst, aged relative to planNow, AT THE
// LISTENER'S OWN LOCATION.
//
// The point matters even in the ordering tests: the fence governs entry (DR-13),
// so a fixture with nowhere to be is fenced out of any burst planned with a
// radius set, and an ordering test would then assert an order over nothing. This
// is what the fence's arrival broke, and it broke loudly, which is the point.
func arrival(id string, c category.Category, sev int, age time.Duration) Arrival {
	return Arrival{ID: id, Category: c, Headline: id + " headline", Subject: "Bonsall, CA",
		Severity: sev, At: planNow.Add(-age),
		Lat: bonsall.Lat, Lon: bonsall.Lon, HasPoint: true}
}

// many is n arrivals of one category, each a little older than the last so the
// within-rung sort has something to be deterministic about.
func many(prefix string, c category.Category, n int) []Arrival {
	out := make([]Arrival, 0, n)
	for i := range n {
		out = append(out, arrival(fmt.Sprintf("%s%02d", prefix, i), c, 50, time.Duration(i)*time.Minute))
	}
	return out
}

// fenceFor is a fence in force, or none. The DR-12 matrix turns on WHICH, not
// on where: its arrivals sit at the listener's own location, so every one of
// them is admitted either way and the only thing the fence changes is the order.
func fenceFor(on bool) Fence {
	if !on {
		return Fence{}
	}
	return bonsall
}

func mustPlan(t *testing.T, arrivals []Arrival, s Settings) Burst {
	t.Helper()
	b, err := Plan(arrivals, s, planNow)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	return b
}

// TestTheShippedOrderReproducesTheRatifiedLadder is DR-10, rung by rung against
// the HUM LEAD's own table (lineup-model.md, "The default order"). The ladder is
// DERIVED from category.ReadRank rather than written out a second time — the
// registry is the single source (NFR-D-6) and the Director must not become a
// fifth list. This test is what makes that derivation checkable.
func TestTheShippedOrderReproducesTheRatifiedLadder(t *testing.T) {
	want := []Rung{
		{CloseBand, category.Emergency}, // — · exempt from Max, above the ladder
		{CloseBand, category.Disasters}, // 2
		{CloseBand, category.Warnings},  // 3
		{CloseBand, category.Watches},   // 4
		{CloseBand, category.Advisories},
		{CloseBand, category.Statements},
		{CloseBand, category.Marine}, // 7
		{RemainingBand, category.Disasters},
		{RemainingBand, category.Warnings},
		{RemainingBand, category.Watches},
		{RemainingBand, category.Advisories},
		{RemainingBand, category.Statements},
		{RemainingBand, category.Marine}, // 13
	}
	got := Ladder()
	if len(got) != len(want) {
		t.Fatalf("the ladder has %d rungs, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rung %d is %v %v, want %v %v", i+1,
				got[i].Band, got[i].Category, want[i].Band, want[i].Category)
		}
	}
	// EMERGENCY ORDERS ARE NEVER DEMOTED. Only Disasters carry the freshness
	// rule, so there is no rung for a remaining emergency and nothing may sort
	// one below a warning.
	for _, r := range got {
		if r.Band == RemainingBand && r.Category == category.Emergency {
			t.Error("the ladder has a rung for a demoted Emergency Order")
		}
	}
}

// TestForecastsNeverEnterABurst is DR-10's other half. An outlook is what might
// happen; the burst is for what is. A forecast stays part of the location
// report, which is most of the broadcast — "never read" means "never a
// takeover" and nothing more.
func TestForecastsNeverEnterABurst(t *testing.T) {
	b := mustPlan(t, []Arrival{
		arrival("f1", category.Forecasts, 90, time.Minute),
		arrival("w1", category.Warnings, 10, time.Minute),
	}, Settings{Max: 5})
	if len(b.Takeover.Refs) != 1 || b.Takeover.Refs[0] != "w1" {
		t.Fatalf("read %v, want only w1", b.Takeover.Refs)
	}
	// And it is not counted as something the listener missed: the notice says
	// "these and N other ALERTS", and a forecast is not one.
	if b.Divert != 0 {
		t.Errorf("divert = %d, want 0 — a forecast is not an alert the burst diverted", b.Divert)
	}
	for _, r := range Ladder() {
		if r.Category == category.Forecasts {
			t.Error("Forecasts has a rung on the ladder")
		}
	}
}

// TestEmergencyOrdersLeadAndSpendTheBudget is DR-11, in the HUM LEAD's own two
// cases at Max 5.
func TestEmergencyOrdersLeadAndSpendTheBudget(t *testing.T) {
	t.Run("one emergency and twenty warnings", func(t *testing.T) {
		in := append(many("e", category.Emergency, 1), many("w", category.Warnings, 20)...)
		b := mustPlan(t, in, Settings{Max: 5})
		if len(b.Takeover.Refs) != 5 {
			t.Fatalf("read %d cards, want 5: %v", len(b.Takeover.Refs), b.Takeover.Refs)
		}
		if b.Takeover.Refs[0] != "e00" {
			t.Errorf("the burst leads with %q, want the emergency order", b.Takeover.Refs[0])
		}
		if b.Divert != 16 {
			t.Errorf("divert = %d, want 16", b.Divert)
		}
	})
	t.Run("seven emergencies and eighteen warnings", func(t *testing.T) {
		in := append(many("e", category.Emergency, 7), many("w", category.Warnings, 18)...)
		b := mustPlan(t, in, Settings{Max: 5})
		if len(b.Takeover.Refs) != 7 {
			t.Fatalf("read %d cards, want all 7 emergency orders: %v", len(b.Takeover.Refs), b.Takeover.Refs)
		}
		for _, r := range b.Takeover.Refs {
			if !strings.HasPrefix(r, "e") {
				t.Errorf("%q was read; when the emergency orders alone exceed Max, everything else is diverted", r)
			}
		}
		if b.Divert != 18 {
			t.Errorf("divert = %d, want 18", b.Divert)
		}
	})
}

// TestAMaxOfNoneStillReadsTheEmergencyOrders. Max bounds the burst; it does not
// silence an evacuation order, which is the one thing that is always read.
func TestAMaxOfNoneStillReadsTheEmergencyOrders(t *testing.T) {
	in := append(many("e", category.Emergency, 2), many("w", category.Warnings, 3)...)
	b := mustPlan(t, in, Settings{Max: 0})
	if got := b.Takeover.Refs; len(got) != 2 || got[0] != "e00" || got[1] != "e01" {
		t.Fatalf("read %v, want both emergency orders and nothing else", got)
	}
	if b.Divert != 3 {
		t.Errorf("divert = %d, want 3", b.Divert)
	}
}

// TestTheDivertCountIsExactlyWhatWasNotRead is DR-14. The number is spoken to
// the listener, so it is the one figure in the burst that must not be
// approximate.
func TestTheDivertCountIsExactlyWhatWasNotRead(t *testing.T) {
	for _, tc := range []struct {
		name              string
		in                []Arrival
		max, read, divert int
	}{
		{"forty arrive, five are read", many("w", category.Warnings, 40), 5, 5, 35},
		{"nothing arrives", nil, 5, 0, 0},
		{"fewer arrive than the Max", many("w", category.Warnings, 3), 5, 3, 0},
		{"exactly the Max arrives", many("w", category.Warnings, 5), 5, 5, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := mustPlan(t, tc.in, Settings{Max: tc.max})
			if len(b.Takeover.Refs) != tc.read {
				t.Errorf("read %d, want %d", len(b.Takeover.Refs), tc.read)
			}
			if b.Divert != tc.divert {
				t.Errorf("divert = %d, want %d", b.Divert, tc.divert)
			}
			if got := len(b.Takeover.Refs) + b.Divert; got != len(tc.in) {
				t.Errorf("read+diverted = %d, want the %d that arrived", got, len(tc.in))
			}
		})
	}
}

// TestDisastersVsWarningsIsConditionalOnTheFence is DR-12's four-cell matrix.
//
// THE NO-FENCE ROWS ARE THE PRIMARY FIXTURE, not the edge case:
// config.TickerRadiusMi defaults to 0 = All (platform/config/config.go:260), so
// every fresh install lands there. The rule is the listener's own — "I care more
// about today's severe thunderstorm warning than a landslide that happened four
// days ago."
func TestDisastersVsWarningsIsConditionalOnTheFence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fenced bool
		age    time.Duration
		first  string
	}{
		{"no fence, a fresh disaster", false, 2 * time.Hour, "d1"},
		{"no fence, a disaster four days ago", false, 96 * time.Hour, "w1"},
		{"a fence, a fresh disaster", true, 2 * time.Hour, "d1"},
		{"a fence, a disaster four days ago", true, 96 * time.Hour, "d1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := mustPlan(t, []Arrival{
				arrival("w1", category.Warnings, 50, time.Hour),
				arrival("d1", category.Disasters, 50, tc.age),
			}, Settings{Max: 5, Fence: fenceFor(tc.fenced)})
			if got := b.Takeover.Refs; len(got) != 2 || got[0] != tc.first {
				t.Errorf("read %v, want %q first", got, tc.first)
			}
		})
	}
}

// TestTheFreshnessBoundaryIsTwentyFourHours (R-5, ratified). A disaster is fresh
// UP TO AND INCLUDING the window; past it, it sorts down into the remaining
// band. It still appears on the tape and in [w] — it merely stops outranking
// live weather.
func TestTheFreshnessBoundaryIsTwentyFourHours(t *testing.T) {
	for _, tc := range []struct {
		name  string
		age   time.Duration
		first string
	}{
		{"a minute inside the window", DisasterFreshWindow - time.Minute, "d1"},
		{"exactly at the window", DisasterFreshWindow, "d1"},
		{"a minute past it", DisasterFreshWindow + time.Minute, "w1"},
		{"dated in the future by a skewed clock", -time.Hour, "d1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := mustPlan(t, []Arrival{
				arrival("w1", category.Warnings, 50, time.Hour),
				arrival("d1", category.Disasters, 50, tc.age),
			}, Settings{Max: 5})
			if got := b.Takeover.Refs; len(got) != 2 || got[0] != tc.first {
				t.Errorf("read %v, want %q first", got, tc.first)
			}
		})
	}
}

// TestWithinARungTheWorstIsReadFirst. Category priority decides the rung;
// inside one rung the order is today's: severity, then recency, then the
// identity so that the sort is TOTAL. A partial order would make the plan
// depend on the arrival order, which is the determinism DR-2 forbids.
func TestWithinARungTheWorstIsReadFirst(t *testing.T) {
	b := mustPlan(t, []Arrival{
		arrival("mild-old", category.Warnings, 10, 4*time.Hour),
		arrival("bad-old", category.Warnings, 90, 4*time.Hour),
		arrival("mild-new", category.Warnings, 10, time.Minute),
		arrival("bad-new", category.Warnings, 90, time.Minute),
	}, Settings{Max: 5})
	want := []string{"bad-new", "bad-old", "mild-new", "mild-old"}
	if got := b.Takeover.Refs; !equal(got, want) {
		t.Errorf("read %v, want %v", got, want)
	}
}

// TestTiesAreBrokenByIdentity — two alerts identical in every sort dimension
// still have exactly one order, and it is the same order every time.
func TestTiesAreBrokenByIdentity(t *testing.T) {
	b := mustPlan(t, []Arrival{
		arrival("zulu", category.Warnings, 50, time.Hour),
		arrival("alpha", category.Warnings, 50, time.Hour),
	}, Settings{Max: 5})
	if got, want := b.Takeover.Refs, []string{"alpha", "zulu"}; !equal(got, want) {
		t.Errorf("read %v, want %v", got, want)
	}
}

// TestThePlanIsDeterministic is DR-2. The same inputs twice produce identical
// lineups, and — the stronger property — the arrival order does not reach the
// output at all.
func TestThePlanIsDeterministic(t *testing.T) {
	in := append(many("w", category.Warnings, 6), many("d", category.Disasters, 4)...)
	in = append(in, many("m", category.Marine, 3)...)
	first := mustPlan(t, in, Settings{Max: 8})
	second := mustPlan(t, in, Settings{Max: 8})
	if !equal(first.Takeover.Refs, second.Takeover.Refs) {
		t.Fatalf("two plans from one input differ:\n%v\n%v", first.Takeover.Refs, second.Takeover.Refs)
	}

	shuffled := make([]Arrival, len(in))
	for i, a := range in { // a fixed reversal, not a random shuffle: a test that
		shuffled[len(in)-1-i] = a // varies run to run cannot be replayed
	}
	third := mustPlan(t, shuffled, Settings{Max: 8})
	if !equal(first.Takeover.Refs, third.Takeover.Refs) {
		t.Errorf("the arrival order reached the plan:\n%v\n%v", first.Takeover.Refs, third.Takeover.Refs)
	}
	if first.Divert != third.Divert {
		t.Errorf("divert differs: %d vs %d", first.Divert, third.Divert)
	}
}

// TestAPlannedCardIsAdmittedAndHasNoWordsYet. The planner's output goes onto the
// rail, and entry to the lineup is admission — so a planned card is already
// past the pre-screen, with its text still to materialise at standby (DR-7).
func TestAPlannedCardIsAdmittedAndHasNoWordsYet(t *testing.T) {
	b := mustPlan(t, many("w", category.Warnings, 3), Settings{Max: 5})
	l := Lineup{}
	for _, c := range []Card{b.Takeover} {
		if c.State != Admitted {
			t.Errorf("%s is at %v, want %v", c.ID, c.State, Admitted)
		}
		if c.Script.Text() != "" {
			t.Errorf("%s already carries words: %q", c.ID, c.Script.Text())
		}
		if c.Slot != BreakingAlert || c.Origin != FromObserver {
			t.Errorf("%s is a %v from %v, want a %v from %v", c.ID, c.Slot, c.Origin, BreakingAlert, FromObserver)
		}
		if !c.Slot.CountsAgainstMax() {
			t.Errorf("%s does not spend the Max; the budget would never run out", c.ID)
		}
		next, err := l.Queue(AlertRail, c)
		if err != nil {
			t.Fatalf("a planned card was refused by the rail: %v", err)
		}
		l = next
	}
	// ONE CARD, whatever the burst holds (MVS-D-77): three alerts compose one
	// takeover, and the rail receives that.
	if got := len(l.Cards(AlertRail)); got != 1 {
		t.Errorf("the rail holds %d cards, want the one takeover", got)
	}
	if got := len(b.Takeover.Refs); got != 3 {
		t.Errorf("the takeover reads %d alerts, want the three that arrived", got)
	}
}

// TestThePlanReadsOnlyWhatArrived — nothing is invented, and nothing is read
// twice. The burst is a snapshot of what came in.
func TestThePlanReadsOnlyWhatArrived(t *testing.T) {
	in := append(many("w", category.Warnings, 4), many("d", category.Disasters, 4)...)
	b := mustPlan(t, in, Settings{Max: 5})
	came := map[string]bool{}
	for _, a := range in {
		came[a.ID] = true
	}
	seen := map[string]bool{}
	for _, r := range b.Takeover.Refs {
		if !came[r] {
			t.Errorf("%q was read but never arrived", r)
		}
		if seen[r] {
			t.Errorf("%q was read twice", r)
		}
		seen[r] = true
	}
}

// TestAMalformedArrivalStopsThePlan. The burst is built once and then spoken;
// a card the lineup would refuse must be found here, where the Director can
// fault it, and not halfway through a takeover.
func TestAMalformedArrivalStopsThePlan(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []Arrival
		s    Settings
	}{
		{"no identity", []Arrival{{Category: category.Warnings, Headline: "h"}}, Settings{Max: 5}},
		{"no headline", []Arrival{{ID: "w1", Category: category.Warnings}}, Settings{Max: 5}},
		{"two under one identity", []Arrival{
			arrival("w1", category.Warnings, 50, time.Hour),
			arrival("w1", category.Disasters, 50, time.Hour),
		}, Settings{Max: 5}},
		{"a negative Max", many("w", category.Warnings, 2), Settings{Max: -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Plan(tc.in, tc.s, planNow); err == nil {
				t.Error("Plan accepted it; want a refusal")
			}
		})
	}
}

// TestTheBandsNameThemselves — the transition log DR-23 asks for names the rung
// a card was read at, and a band with no name makes that line unreadable.
func TestTheBandsNameThemselves(t *testing.T) {
	if got, want := CloseBand.String(), "CLOSE"; got != want {
		t.Errorf("CloseBand = %q, want %q", got, want)
	}
	if got, want := RemainingBand.String(), "REMAINING"; got != want {
		t.Errorf("RemainingBand = %q, want %q", got, want)
	}
	for _, bad := range []Band{-1, numBands, 1 << 20} {
		if got := bad.String(); got != "" {
			t.Errorf("Band(%d).String() = %q, want empty", bad, got)
		}
	}
}

// TestPlannedBurstGolden is DR-2's golden: one fixed arrival set spanning every
// readable category, both bands and the emergency overrun boundary, rendered as
// the rung each card was read at.
//
// A golden catches what no single assertion above would: a category quietly
// re-ranked, a band boundary moved, a tie broken the other way. The text of the
// read is deliberately absent — it does not exist until standby, and pinning
// the headline would make every wording change look like a reordering.
func TestPlannedBurstGolden(t *testing.T) {
	in := []Arrival{
		arrival("emerg-evac", category.Emergency, 99, 10*time.Minute),
		arrival("quake-fresh", category.Disasters, 70, 3*time.Hour),
		arrival("slide-stale", category.Disasters, 80, 96*time.Hour),
		arrival("tstorm", category.Warnings, 60, 20*time.Minute),
		arrival("flood", category.Warnings, 60, 2*time.Hour),
		arrival("tornado-watch", category.Watches, 55, time.Hour),
		arrival("heat", category.Advisories, 30, 5*time.Hour),
		arrival("spec-stmt", category.Statements, 20, 30*time.Minute),
		arrival("small-craft", category.Marine, 40, 45*time.Minute),
		arrival("outlook", category.Forecasts, 10, time.Hour),
	}
	b := mustPlan(t, in, Settings{Max: 6})

	var out strings.Builder
	fmt.Fprintf(&out, "max=6 fenced=false arrivals=%d read=%d divert=%d\n\n", len(in), len(b.Takeover.Refs), b.Divert)
	for i, r := range b.Takeover.Refs {
		a := in[0]
		for _, cand := range in {
			if cand.ID == r {
				a = cand
			}
		}
		band := bandOf(a.Category, Settings{Max: 6}, a.At, planNow)
		fmt.Fprintf(&out, "%2d. rung=%2d %-9s %-18s %-14s sev=%d\n",
			i+1, rungOf(band, a.Category), band, category.Of(a.Category).Bucket, r, a.Severity)
	}
	got := out.String()

	path := filepath.Join("testdata", "plan.golden")
	if *updatePlan {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Log("golden re-captured; review the diff line by line")
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no golden yet: run with -update-plan (%v)", err)
	}
	if got != string(want) {
		t.Errorf("the planned burst changed:\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}
