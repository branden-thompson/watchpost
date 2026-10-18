package main

import (
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// Places the tests draw from: a tower in west Texas, far from every
// continent's reference point; receivers about 100 km from it and one 20 km
// off; a transmitter in Frankfurt's region.
var (
	tower      = ionomaps.LatLon{Lat: 31.0, Lon: -100.0}
	rxNorth    = ionomaps.LatLon{Lat: 31.9, Lon: -100.0}
	rxEast     = ionomaps.LatLon{Lat: 31.0, Lon: -98.95}
	rxSouth    = ionomaps.LatLon{Lat: 30.1, Lon: -100.0}
	rxNear     = ionomaps.LatLon{Lat: 31.18, Lon: -100.0}
	frankfurt  = ionomaps.LatLon{Lat: 50.0, Lon: 8.5}
	scoredHour = time.Date(2026, 9, 29, 3, 35, 0, 0, time.UTC)
)

// heard is a spot row for the tests.
func heard(tx string, txAt ionomaps.LatLon, rx string, rxAt ionomaps.LatLon, band ionomaps.Band, hour time.Time, best float64) spot {
	return spot{Tx: tx, Rx: rx, TxAt: txAt, RxAt: rxAt, Band: band, Hour: hour.Truncate(time.Hour), N: 1, Best: best}
}

// TestEachPairIsGivenItsOutcome is D-218 (A2) and D-219's reach: heard at
// +13 dB normalised is open, heard only below it weak, a pair both monitored
// and unheard unheard; the ground wave's pairs are not drawn; a US tower's
// served places are M1's, a continent's region's transmitters M4's.
func TestEachPairIsGivenItsOutcome(t *testing.T) {
	b := ionomaps.Band40m
	h := scoredHour
	spots := []spot{
		heard("T", tower, "N", rxNorth, b, h, 15),                  // open
		heard("T", tower, "E", rxEast, b, h, 10),                   // weak
		heard("O", frankfurt, "S", rxSouth, b, h, 2),               // S monitored, so T-S unheard; O-S weak, M4's
		heard("T", tower, "C", rxNear, b, h, 30),                   // the ground wave's: not drawn
		heard("T", tower, "X", rxSouth, b, h.Add(2*time.Hour), 30), // past the window
		heard("T", tower, "Y", rxSouth, ionomaps.Band20m, h, 30),   // another band
	}
	got := map[[3]string]outcome{}
	key := func(s scenario) [3]string {
		names := map[ionomaps.LatLon]string{tower: "T", rxNorth: "N", rxEast: "E", rxSouth: "S", rxNear: "C", frankfurt: "O"}
		return [3]string{s.Metric.String(), names[s.Origin], names[s.Target]}
	}
	r := reach{SkyKm: defaultSkyKm, ServedKm: defaultServedKm, RegionKm: defaultRegionKm, NearKm: defaultNearKm}
	windowOf(spots, h, b).classify(h, b, r, func(s scenario) { got[key(s)] = s.Outcome })
	want := map[[3]string]outcome{
		{"M1", "T", "N"}: open, {"M1", "T", "E"}: weak, {"M1", "T", "S"}: unheard,
		{"M4", "S", "O"}: weak, {"M4", "N", "O"}: unheard, {"M4", "E", "O"}: unheard, {"M4", "C", "O"}: unheard,
	}
	for k, o := range want {
		if g, drawn := got[k]; !drawn || g != o {
			t.Errorf("%v: %v (drawn %t); want %v", k, g, drawn, o)
		}
	}
	if len(got) != len(want) {
		t.Errorf("drawn %v; want only %v", got, want)
	}
}

// TestTheStrongestSpotInTheWindowDecides is the ±1 h window: the strongest
// spot in the hours either side of the clock's decides, one two hours off
// counts for nothing.
func TestTheStrongestSpotInTheWindowDecides(t *testing.T) {
	b := ionomaps.Band20m
	h := scoredHour
	w := windowOf([]spot{
		heard("T", tower, "N", rxNorth, b, h.Add(-time.Hour), 10),
		heard("T", tower, "N", rxNorth, b, h.Add(time.Hour), 14),
		heard("T", tower, "E", rxEast, b, h.Add(-time.Hour), 9),
		heard("T", tower, "E", rxEast, b, h.Add(-2*time.Hour), 20),
	}, h, b)
	if o := w.outcomeOf(pair{"T", "N"}); o != open {
		t.Errorf("heard at 10 then 14 dB in the window: %v; want open", o)
	}
	if o := w.outcomeOf(pair{"T", "E"}); o != weak {
		t.Errorf("heard at 9 dB in the window, 20 dB two hours before: %v; want weak", o)
	}
	if o := w.outcomeOf(pair{"T", "S"}); o != unheard {
		t.Errorf("a pair never heard: %v; want unheard", o)
	}
	at13 := windowOf([]spot{heard("T", tower, "N", rxNorth, b, h, voiceDB)}, h, b)
	if o := at13.outcomeOf(pair{"T", "N"}); o != open {
		t.Errorf("heard at exactly +13 dB: %v; want open (D-91: at least +13 dB)", o)
	}
}

// TestTheDrawIsBalancedSoAConstantScoresHalf is D-219 (B1): each stratum
// as many closed as open, at most k, the closed half weak and half unheard
// as far as each goes - so a constant answer scores 50 % - every pair
// counted for the base rate, and the draw fixed by its seed.
func TestTheDrawIsBalancedSoAConstantScoresHalf(t *testing.T) {
	feed := func(seed uint64) (*drawer, []scenario) {
		d := newDrawer(10, seed)
		for o, n := range []int{open: 30, weak: 5, unheard: 50} { // in a fixed order: the draw is fixed by its seed and the order offered
			for j := range n {
				d.offer(scenario{Metric: m1, At: scoredHour, Band: ionomaps.Band40m, Km: float64(j), Outcome: outcome(o)})
			}
		}
		for j := range 20 {
			d.offer(scenario{Metric: m4, At: scoredHour, Band: ionomaps.Band20m, Km: float64(j), Outcome: weak})
		}
		for j := range 3 {
			d.offer(scenario{Metric: m4, At: scoredHour, Band: ionomaps.Band20m, Km: float64(j), Outcome: open})
		}
		set, err := d.set()
		if err != nil {
			t.Fatal(err)
		}
		return d, set
	}
	d, set := feed(19)
	var n [metrics][outcomes]int
	var constant [metrics]tally
	for _, s := range set {
		n[s.Metric][s.Outcome]++
		constant[s.Metric].add(s.Outcome, true, false)
	}
	if n[m1] != [outcomes]int{10, 5, 5} || n[m4] != [outcomes]int{3, 3, 0} {
		t.Errorf("drawn by outcome: M1 %v, M4 %v; want [10 5 5] and [3 3 0]", n[m1], n[m4])
	}
	for m := range metrics {
		if total, agree := constant[m].total(); 2*agree != total {
			t.Errorf("%v: \"always reaches\" agrees %d of %d; want half", m, agree, total)
		}
	}
	if c := d.counts[base{m1, ionomaps.Band40m}]; c == nil || *c != [outcomes]int{30, 5, 50} {
		t.Errorf("the base rate's counts: %v; want every pair, [30 5 50]", c)
	}
	_, again := feed(19)
	_, other := feed(20)
	same, differs := len(again) == len(set), false
	for i := range min(len(set), len(again), len(other)) {
		same = same && again[i] == set[i]
		differs = differs || other[i] != set[i]
	}
	if !same || !differs {
		t.Errorf("the draw: the same seed the same set %t, another seed another %t", same, differs)
	}
}
