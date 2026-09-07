package lineup

import (
	"math"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/geo"
)

// The listener, and the two places the HUM LEAD used to state the rule.
var (
	bonsall   = Fence{RadiusMi: 50, Lat: 33.2886, Lon: -117.2247, HasOrigin: true}
	losAngels = [2]float64{34.0522, -118.2437}
	fairbanks = [2]float64{64.8378, -147.7164}
)

// milesFrom is the distance a test claims, measured the same way the fence
// measures it. A test that asserted "~120 mi" without checking would be
// asserting its own arithmetic.
func milesFrom(f Fence, at [2]float64) float64 {
	return geo.HaversineKM(f.Lat, f.Lon, at[0], at[1]) / kmPerMi
}

// quake is a disaster arrival at a point, carrying the reach its magnitude buys.
func quake(id string, mag float64, at [2]float64) Arrival {
	return Arrival{ID: id, Category: category.Disasters, Headline: id + " headline",
		Subject: "offshore", Severity: 90, At: planNow.Add(-time.Hour),
		Lat: at[0], Lon: at[1], HasPoint: true, ReachMi: QuakeReachMi(mag)}
}

// ruledAdmitMi is the distance the HUM LEAD's admit-case names: "an M7.5 quake
// in Los Angeles (~120 mi away)".
//
// IT IS AN ILLUSTRATION, NOT A THRESHOLD (ratified 2026-09-02): the two cases
// were given to show "the correlation between distance and urgency /
// importance", not to set a boundary at 120 miles. It is still asserted, because
// a scale that cannot reproduce the example the rule was explained with is not
// encoding the rule — but a change that moved it would be a change to the shape,
// not a broken contract, and the failure message says so.
//
// It is asserted as a NUMBER rather than inferred from the Los Angeles fixture,
// because the two are not the same thing: Los Angeles measures ~79 mi from
// Bonsall by great circle, and ~120 is what a person means by how far away Los
// Angeles is.
const ruledAdmitMi = 120

// at is a point ruledAdmitMi-style distances can be built from: due north of the
// listener by the given miles, in the same arithmetic the fence uses.
func northOf(f Fence, mi float64) [2]float64 {
	const degPerKm = 1 / 111.19492664455873
	return [2]float64{f.Lat + mi*kmPerMi*degPerKm, f.Lon}
}

// TestTheFixturesAreTheDistancesTheyClaim is the fixture-validity control
// (anti-vacuity rule 1), in the form this package can state it. Every case below
// turns on a distance, so the distances are measured HERE, before anything is
// asserted about admission — a fixture whose coordinates had drifted would
// otherwise make every test below pass for the wrong reason.
func TestTheFixturesAreTheDistancesTheyClaim(t *testing.T) {
	if got := milesFrom(bonsall, northOf(bonsall, ruledAdmitMi)); math.Abs(got-ruledAdmitMi) > 0.5 {
		t.Errorf("the ruling's admit-case fixture measures %.1f mi, want %d", got, ruledAdmitMi)
	}
	// Both fixtures must be OUTSIDE the fence, or the exception under test is
	// never reached and every case below passes on the radius alone.
	for name, d := range map[string]float64{
		"the ruling's ~120 mi": ruledAdmitMi,
		"Los Angeles":          milesFrom(bonsall, losAngels),
		"Fairbanks":            milesFrom(bonsall, fairbanks),
	} {
		if d <= bonsall.RadiusMi {
			t.Errorf("%s is %.0f mi away, inside the %.0f mi fence — the exception would never be reached",
				name, d, bonsall.RadiusMi)
		}
	}
	if got := milesFrom(bonsall, fairbanks); got < 2000 {
		t.Errorf("Fairbanks measures %.0f mi from the listener, want the thousands the ruling assumes", got)
	}
}

// TestASignificantDisasterCarriesItsOwnReach is DR-13, in the HUM LEAD's own two
// cases: "I care about an M7.5 quake in Los Angeles (~120 mi away). I don't
// think I'd notice an M9.0 in Fairbanks, Alaska."
func TestASignificantDisasterCarriesItsOwnReach(t *testing.T) {
	ruled := quake("m75-at-120mi", 7.5, northOf(bonsall, ruledAdmitMi))
	inLA := quake("m75-la", 7.5, losAngels)
	far := quake("m90-fairbanks", 9.0, fairbanks)

	// THE CONTROL FOR THE REFUSE-CASE. Before asserting that the M9.0 is kept
	// out, assert that the same quake at the listener's own doorstep is let in.
	// Without this, "not admitted" would also be true of an arrival the fence
	// rejects for being malformed — which is how a test proves nothing.
	atHome := far
	atHome.Lat, atHome.Lon = bonsall.Lat, bonsall.Lon
	if !bonsall.Admits(atHome) {
		t.Fatal("the M9.0 fixture is refused even at the listener's own location; nothing below would mean anything")
	}

	if !bonsall.Admits(ruled) {
		t.Errorf("the M7.5 at the ruling's %d mi was fenced out; its own reach is %.0f mi",
			ruledAdmitMi, ruled.ReachMi)
	}
	if !bonsall.Admits(inLA) {
		t.Errorf("the M7.5 in Los Angeles (%.0f mi) was fenced out; its own reach is %.0f mi",
			milesFrom(bonsall, losAngels), inLA.ReachMi)
	}
	if bonsall.Admits(far) {
		t.Errorf("the M9.0 at %.0f mi was admitted; its own reach is only %.0f mi",
			milesFrom(bonsall, fairbanks), far.ReachMi)
	}
}

// TestReachGrowsWithMagnitudeAndStopsBelowTheStrongThreshold. The scale is one
// function with two constants (BD-6); this pins its shape rather than its
// arithmetic — that a bigger quake reaches further, that the growth is real
// enough to matter, and that an ordinary quake buys no exception at all.
func TestReachGrowsWithMagnitudeAndStopsBelowTheStrongThreshold(t *testing.T) {
	for _, mag := range []float64{0, 1.5, 4.9, QuakeReachFrom - 0.1} {
		if got := QuakeReachMi(mag); got != 0 {
			t.Errorf("an M%.1f buys %.0f mi of reach, want none", mag, got)
		}
	}
	last := 0.0
	for mag := QuakeReachFrom; mag <= 9.5; mag += 0.5 {
		got := QuakeReachMi(mag)
		if got <= last {
			t.Errorf("an M%.1f reaches %.0f mi, no further than the magnitude below it (%.0f mi)", mag, got, last)
		}
		last = got
	}
	// The ruling's two cases, stated against the SCALE rather than only through
	// the fence, so a change to either constant fails here first and says why.
	if got := QuakeReachMi(7.5); got < ruledAdmitMi {
		t.Errorf("an M7.5 reaches %.0f mi and no longer covers the ruling's illustration of ~%d mi; if the scale's shape was meant to change, this is the line that says so",
			got, ruledAdmitMi)
	}
	if got, fb := QuakeReachMi(9.0), milesFrom(bonsall, fairbanks); got >= fb {
		t.Errorf("an M9.0 reaches %.0f mi, far enough to pull in Fairbanks at %.0f mi", got, fb)
	}
	// A magnitude the feed could never issue must not reopen the fence. USGS
	// clamps its field to 12, which is a bound on bad data rather than a
	// magnitude that can occur, and doubling per point that would reach far
	// enough to admit the ruling's own counter-example.
	if got, fb := QuakeReachMi(12), milesFrom(bonsall, fairbanks); got >= fb {
		t.Errorf("an M12 reaches %.0f mi, enough to pull in Fairbanks at %.0f mi", got, fb)
	}
	if QuakeReachMi(12) != QuakeReachMi(quakeReachTo) {
		t.Error("the magnitude ceiling is not being applied")
	}
}

// TestOnlyADisasterCarriesReach. The exception is for a significant DISASTER,
// because such a disaster has proximal effects. A warning a thousand miles away
// is still a warning a thousand miles away, however severe.
func TestOnlyADisasterCarriesReach(t *testing.T) {
	for c := category.Category(0); c < category.Count; c++ {
		a := quake("x", 9.0, losAngels)
		a.Category = c
		if got, want := bonsall.Admits(a), c == category.Disasters; got != want {
			t.Errorf("a %v at %.0f mi with %.0f mi of reach: admitted = %v, want %v",
				category.Of(c).Bucket, milesFrom(bonsall, losAngels), a.ReachMi, got, want)
		}
	}
}

// TestAnEmergencyOrderIsNotExemptFromTheRadius (R-2). "I don't care about a fire
// evac order in Los Angeles if I'm in San Diego — but the AIR QUALITY HAZARD
// STATEMENT for San Diego as a result of that fire is more relevant to me." The
// consequence that reaches the listener arrives as its own local alert, inside
// the fence, so the design never has to model the link.
func TestAnEmergencyOrderIsNotExemptFromTheRadius(t *testing.T) {
	evac := Arrival{ID: "evac", Category: category.Emergency, Headline: "Evacuation Order",
		Subject: "Los Angeles", Severity: 99, At: planNow,
		Lat: losAngels[0], Lon: losAngels[1], HasPoint: true}
	if bonsall.Admits(evac) {
		t.Error("an evacuation order beyond the fence was admitted; emergency orders are exempt from the Max, not the radius")
	}
	near := evac
	near.Lat, near.Lon = bonsall.Lat, bonsall.Lon
	if !bonsall.Admits(near) {
		t.Error("an evacuation order at the listener's own location was fenced out")
	}
}

// TestNoFenceAdmitsEverything. THE DEFAULT PATH: config.TickerRadiusMi defaults
// to 0 = All, so every fresh install is here.
func TestNoFenceAdmitsEverything(t *testing.T) {
	all := Fence{}
	if all.InForce() {
		t.Error("a zero radius reads as a fence in force; 0 means All")
	}
	for _, a := range []Arrival{
		quake("far", 9.0, fairbanks),
		{ID: "zone", Category: category.Warnings, Headline: "h", Subject: "s"},
	} {
		if !all.Admits(a) {
			t.Errorf("%s was fenced out with no fence set", a.ID)
		}
	}
}

// TestAFenceWithNoOriginAdmitsNothing. Today's rule, unchanged: filtered with no
// default location set shows NOTHING, rather than silently falling back to the
// global stack the UI says is scoped away.
func TestAFenceWithNoOriginAdmitsNothing(t *testing.T) {
	homeless := Fence{RadiusMi: 50}
	if !homeless.InForce() {
		t.Fatal("a 50 mi radius does not read as a fence in force")
	}
	if homeless.Admits(quake("m90", 9.0, losAngels)) {
		t.Error("a fence with nowhere to measure from admitted something")
	}
}

// TestAPointlessAlertIsAdmittedOnlyByTheTrackedTie. A zone-only NWS alert has no
// point to measure, and today it reaches a scoped surface only by being one the
// app is already tracking at a watched location (app/severe.go:scopeEvents).
// The fence must say the same thing, or the two surfaces disagree about one
// hazard — which is the failure that rule exists to prevent.
func TestAPointlessAlertIsAdmittedOnlyByTheTrackedTie(t *testing.T) {
	base := Arrival{ID: "zone", Category: category.Warnings, Headline: "h", Subject: "s", At: planNow}
	if bonsall.Admits(base) {
		t.Error("an untracked zone-only alert was admitted through a fence")
	}
	tracked := base
	tracked.Tracked = true
	if !bonsall.Admits(tracked) {
		t.Error("a tracked zone-only alert was fenced out")
	}
}

// TestTheFenceAgreesWithTheTapeAboutEveryDistance.
//
// scopeEvents warns that "the two surfaces cannot disagree about one hazard",
// and this is that promise made checkable. globalfeed.WithinMiles decides what
// reaches the tape and [w]; this fence decides what reaches the burst. They are
// separate code because platform/ cannot import domains/*, so the only thing
// keeping them together is that they compute the same expression.
func TestTheFenceAgreesWithTheTapeAboutEveryDistance(t *testing.T) {
	const degPerKm = 1 / 111.19492664455873
	for _, mi := range []float64{0, 1, 25, 49, 49.999, 50, 50.001, 51, 75, 500} {
		at := [2]float64{bonsall.Lat + mi*kmPerMi*degPerKm, bonsall.Lon}
		a := Arrival{ID: "edge", Category: category.Warnings, Headline: "h", Subject: "s",
			Lat: at[0], Lon: at[1], HasPoint: true}
		want := geo.HaversineKM(bonsall.Lat, bonsall.Lon, at[0], at[1]) <= bonsall.RadiusMi*kmPerMi
		if got := bonsall.Admits(a); got != want {
			t.Errorf("at ~%.3f mi (measured %.4f): admitted = %v, want %v",
				mi, milesFrom(bonsall, at), got, want)
		}
	}
}

// TestAnAlertExactlyOnTheRadiusIsInside pins the boundary's DIRECTION, which the
// test above cannot: it compares the fence against the same expression, so
// flipping both to `<` together keeps them agreeing and the mutant survives.
//
// The difficulty is that no lat/lon fixture lands exactly on a radius in
// floating point. So the fence is built FROM the fixture instead — the radius is
// the measured distance — and the test asserts that the round trip through miles
// is exact before it asserts anything about admission. If that control ever
// stops holding, this fails loudly rather than silently pinning nothing.
func TestAnAlertExactlyOnTheRadiusIsInside(t *testing.T) {
	at := [2]float64{bonsall.Lat + 0.5, bonsall.Lon}
	km := geo.HaversineKM(bonsall.Lat, bonsall.Lon, at[0], at[1])
	edge := Fence{RadiusMi: km / kmPerMi, Lat: bonsall.Lat, Lon: bonsall.Lon, HasOrigin: true}
	if edge.RadiusMi*kmPerMi != km {
		t.Fatalf("the fixture does not sit exactly on the radius in the fence's own arithmetic (%.17g vs %.17g); this test would pin nothing",
			edge.RadiusMi*kmPerMi, km)
	}
	a := Arrival{ID: "edge", Category: category.Warnings, Headline: "h", Subject: "s",
		Lat: at[0], Lon: at[1], HasPoint: true}
	if !edge.Admits(a) {
		t.Errorf("an alert exactly on the %.3f mi radius was fenced out; the boundary is inclusive, as globalfeed.WithinMiles is",
			edge.RadiusMi)
	}
}

// TestAnOrdinaryQuakeBuysNoExceptionEvenAgainstATightFence.
//
// The threshold below which a quake carries no reach is invisible against a
// 50 mi fence — an M5.0's reach would be 23 mi, which is inside the radius
// anyway, so removing the rule changes nothing there. It is visible against a
// SMALL service area, which is the case a Broadcaster station actually has. m83
// survived until this existed.
func TestAnOrdinaryQuakeBuysNoExceptionEvenAgainstATightFence(t *testing.T) {
	const degPerKm = 1 / 111.19492664455873
	tight := Fence{RadiusMi: 10, Lat: bonsall.Lat, Lon: bonsall.Lon, HasOrigin: true}
	at := [2]float64{tight.Lat + 20*kmPerMi*degPerKm, tight.Lon}
	if d := milesFrom(tight, at); d < 19.5 || d > 20.5 {
		t.Fatalf("the fixture measures %.1f mi, not the 20 it is meant to", d)
	}
	ordinary := quake("m50", 5.0, at)
	if ordinary.ReachMi != 0 {
		t.Errorf("an M5.0 carries %.1f mi of reach; below M%.1f a quake carries none", ordinary.ReachMi, QuakeReachFrom)
	}
	if tight.Admits(ordinary) {
		t.Error("an ordinary M5.0 at 20 mi crossed a 10 mi fence; the exception is for a SIGNIFICANT disaster")
	}
	// The control: the same quake one notch above the threshold does cross, so
	// the refusal above is the threshold and not the fixture.
	strong := quake("m65", 6.5, at)
	if !tight.Admits(strong) {
		t.Errorf("an M6.5 with %.0f mi of reach did not cross a 10 mi fence at 20 mi; the refusal above proves nothing",
			strong.ReachMi)
	}
}

// TestTheFenceFiltersWhatReachesTheLineupAtAll is DR-13 through the real entry
// point: the burst is planned from what got past the fence, and nothing else.
func TestTheFenceFiltersWhatReachesTheLineupAtAll(t *testing.T) {
	in := []Arrival{
		quake("m75-la", 7.5, losAngels),        // in, by its own reach
		quake("m90-fairbanks", 9.0, fairbanks), // out
		{ID: "local", Category: category.Warnings, Headline: "h", Subject: "s", Severity: 50,
			At: planNow, Lat: bonsall.Lat, Lon: bonsall.Lon, HasPoint: true},
	}
	b := mustPlan(t, in, Settings{Max: 5, Fence: bonsall})
	if got, want := b.Takeover.Refs, []string{"m75-la", "local"}; !equal(got, want) {
		t.Fatalf("read %v, want %v", got, want)
	}
	// A FENCED-OUT ALERT IS NOT DIVERTED. The count tells the listener what they
	// can go and read; [w] is scoped by the same radius, so counting something
	// the fence removed would point them at a page that does not have it.
	if b.Divert != 0 {
		t.Errorf("divert = %d, want 0 — the M9.0 never entered the listener's world", b.Divert)
	}
}

// TestTheFenceIsTheOnlyCarrierOfWhetherOneIsInForce. The ordering rule (DR-12)
// asks whether a radius is set; so does admission. One question, one answer —
// a separate boolean beside the radius could disagree with it, which is the
// shape of every rule this release has had to un-split.
func TestTheFenceIsTheOnlyCarrierOfWhetherOneIsInForce(t *testing.T) {
	stale := quake("d1", 4.0, [2]float64{bonsall.Lat, bonsall.Lon}) // no reach, at home
	stale.At = planNow.Add(-96 * time.Hour)
	warn := Arrival{ID: "w1", Category: category.Warnings, Headline: "h", Subject: "s", Severity: 50,
		At: planNow.Add(-time.Hour), Lat: bonsall.Lat, Lon: bonsall.Lon, HasPoint: true}

	fenced := mustPlan(t, []Arrival{warn, stale}, Settings{Max: 5, Fence: bonsall})
	if got := fenced.Takeover.Refs; got[0] != "d1" {
		t.Errorf("with a fence in force the burst leads with %q, want the disaster", got[0])
	}
	open := mustPlan(t, []Arrival{warn, stale}, Settings{Max: 5})
	if got := open.Takeover.Refs; got[0] != "w1" {
		t.Errorf("with no fence the burst leads with %q, want the warning", got[0])
	}
}

// TestAnArrivalWithNoRealPointIsRefusedRatherThanSilentlyFencingEverythingOut.
//
// NaN compares false against everything, so an unparseable coordinate would
// fail the radius test AND the significance exception — the hazard refused for
// a reason nobody could see. The invariant makes the refusal deliberate.
func TestAnArrivalWithNoRealPointIsRefusedRatherThanSilentlyFencingEverythingOut(t *testing.T) {
	f := Fence{RadiusMi: 50, HasOrigin: true, Lat: 33.2887, Lon: -117.2179}
	sane := Arrival{ID: "a", HasPoint: true, Lat: 33.3, Lon: -117.2, Category: category.Warnings}
	if !f.Admits(sane) {
		t.Fatal("a warning inside the radius must be admitted; the fixture is wrong")
	}
	broken := sane
	broken.Lat = math.NaN()
	if f.Admits(broken) {
		t.Error("an arrival whose distance is not a number was admitted")
	}
}
