package main

// scenarios.go - the scenarios: each window's pairs given their outcome by
// D-218 (A2), and the set drawn from them by D-219 (B1).

import (
	"math/rand/v2"
	"slices"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/propanswers"
)

// voiceDB is the reference circuit's SNR (D-73, D-91): +13 dB in 2.5 kHz,
// WSPR's own bandwidth, normalised to 100 W.
const voiceDB = 13

// The reach the scenarios are drawn within, until ruled otherwise: M1's
// place is a receiver past the ground wave (PLAN's 50 km, `dry-run.md`) and
// within the largest service radius ruled (100 mi, `platform/config`); M4's
// target a transmitter within 1000 km of a continent's reference point; M4's
// listener answered with the near-vertical radius's default (400 km).
const (
	defaultSkyKm    = 50
	defaultServedKm = 161
	defaultRegionKm = 1000
	defaultNearKm   = 400
)

// reach is where the scenarios are drawn and answered.
type reach struct {
	SkyKm, ServedKm, RegionKm, NearKm float64
}

// metric is the metric a scenario is for.
type metric uint8

// The metrics.
const (
	m1 metric = iota // the tower and a place in its served area
	m4               // a US listener and a transmitter in a continent's region
	metrics
)

// String is the metric's name.
func (m metric) String() string {
	if m >= metrics {
		return "?"
	}
	return [...]string{"M1", "M4"}[m]
}

// outcome is what the spots say of a pair in a window.
type outcome uint8

// The outcomes: open, and D-218's two kinds of closed.
const (
	open    outcome = iota // heard at +13 dB or better, normalised to 100 W
	weak                   // heard, never at +13 dB (A1)
	unheard                // both monitored on the band in the window, neither heard by the other (A2)
	outcomes
)

// String is the outcome's name.
func (o outcome) String() string {
	if o >= outcomes {
		return "?"
	}
	return [...]string{"open", "weak", "unheard"}[o]
}

// scenario is one question and the spots' answer: M1's origin is the tower
// and its target the place; M4's origin is the listener and its target the
// transmitter.
type scenario struct {
	Metric         metric
	At             time.Time // the answer's clock
	Band           ionomaps.Band
	Origin, Target ionomaps.LatLon
	Km             float64
	Outcome        outcome
}

// pair is a transmitter's locator and a receiver's.
type pair struct{ tx, rx string }

// window is one band's spots in a scored hour's window: the strongest each
// pair was heard at, and where each transmitter heard and each receiver
// hearing is.
type window struct {
	heard map[pair]float64
	tx    map[string]ionomaps.LatLon
	rx    map[string]ionomaps.LatLon
}

// windowOf gathers the spots of one band in the window: the hours the clock
// is in, the one before and the one after (±1 h, M1 and M4).
func windowOf(spots []spot, at time.Time, band ionomaps.Band) window {
	w := window{heard: map[pair]float64{}, tx: map[string]ionomaps.LatLon{}, rx: map[string]ionomaps.LatLon{}}
	if len(spots) == 0 {
		return w
	}
	if band < ionomaps.Band160m || band > ionomaps.Band10m {
		return w // not a band go-ionomaps answers: no pair is drawn on it
	}
	hour := at.Truncate(time.Hour)
	for _, s := range spots {
		in := s.Band == band && !s.Hour.Before(hour.Add(-time.Hour)) && !s.Hour.After(hour.Add(time.Hour))
		if !in {
			continue
		}
		k := pair{s.Tx, s.Rx}
		if best, seen := w.heard[k]; !seen || s.Best > best {
			w.heard[k] = s.Best
		}
		if _, seen := w.tx[s.Tx]; !seen {
			w.tx[s.Tx] = s.TxAt
		}
		if _, seen := w.rx[s.Rx]; !seen {
			w.rx[s.Rx] = s.RxAt
		}
	}
	return w
}

// outcomeOf is a monitored pair's outcome in the window.
func (w window) outcomeOf(k pair) outcome {
	best, heard := w.heard[k]
	if !heard {
		return unheard
	}
	if best >= voiceDB {
		return open
	}
	return weak
}

// inRegion reports whether a place is within a continent's region.
func inRegion(p ionomaps.LatLon, km float64) bool {
	if !(km > 0) {
		return false
	}
	if !(p.Lat >= -90 && p.Lat <= 90) {
		return false // off the globe: in no region
	}
	for _, c := range propanswers.Continents {
		if geo.HaversineKM(p.Lat, p.Lon, c.At.Lat, c.At.Lon) <= km {
			return true
		}
	}
	return false
}

// classify gives each monitored pair of the window its scenario, M1's and
// M4's, in a fixed order: the transmitters' locators sorted, then the
// receivers'. Both ends are monitored by construction: a transmitter in the
// window was heard by someone on the band, a receiver heard someone.
func (w window) classify(at time.Time, band ionomaps.Band, r reach, emit func(scenario)) {
	if emit == nil || !(r.SkyKm >= 0) {
		return
	}
	if len(w.tx) == 0 || len(w.rx) == 0 {
		return // nothing heard in the window: no pair is monitored
	}
	txs, rxs := sortedKeys(w.tx), sortedKeys(w.rx)
	for _, t := range txs {
		tp := w.tx[t]
		tower, region := inUS(tp), inRegion(tp, r.RegionKm)
		if !tower && !region {
			continue
		}
		for _, l := range rxs {
			lp := w.rx[l]
			km := geo.HaversineKM(tp.Lat, tp.Lon, lp.Lat, lp.Lon)
			if km < r.SkyKm {
				continue // the ground wave's, not the ionosphere's
			}
			o := w.outcomeOf(pair{t, l})
			if tower && km <= r.ServedKm {
				emit(scenario{Metric: m1, At: at, Band: band, Origin: tp, Target: lp, Km: km, Outcome: o})
			}
			if region {
				emit(scenario{Metric: m4, At: at, Band: band, Origin: lp, Target: tp, Km: km, Outcome: o})
			}
		}
	}
}

// sortedKeys are a map's keys in order.
func sortedKeys(m map[string]ionomaps.LatLon) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// stratum is B1's stratum: the metric, the band and the UTC hour - the
// hour -1 when the strata are by band alone.
type stratum struct {
	Metric metric
	Band   ionomaps.Band
	Hour   int
}

// base is a metric's band, which the base rate is counted by.
type base struct {
	Metric metric
	Band   ionomaps.Band
}

// reservoir is a uniform sample of at most k of what it was offered.
type reservoir struct {
	offered int
	kept    []scenario
}

// keep puts a scenario to the reservoir (Vitter's algorithm R).
func (r *reservoir) keep(s scenario, k int, rng *rand.Rand) {
	if k < 1 {
		return // a reservoir of nothing keeps nothing
	}
	if rng == nil {
		return
	}
	r.offered++
	if len(r.kept) < k {
		r.kept = append(r.kept, s)
		return
	}
	if j := rng.IntN(r.offered); j < k {
		r.kept[j] = s
	}
}

// drawer draws B1's set as the scenarios stream past: per stratum a sample
// of each outcome, and per metric's band how many of each there were.
type drawer struct {
	k      int
	byHour bool
	rng    *rand.Rand
	strata map[stratum]*[outcomes]reservoir
	counts map[base]*[outcomes]int
}

// newDrawer is a drawer keeping at most k of each outcome a stratum, its
// strata by band and UTC hour (D-219) or, byHour false, by band alone, its
// draws fixed by the seed.
func newDrawer(k int, seed uint64, byHour bool) *drawer {
	return &drawer{k: max(k, 1), byHour: byHour, rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		strata: map[stratum]*[outcomes]reservoir{}, counts: map[base]*[outcomes]int{}}
}

// offer counts a scenario and puts it to its stratum's sample.
func (d *drawer) offer(s scenario) {
	if s.Outcome >= outcomes {
		return
	}
	if s.Metric >= metrics {
		return
	}
	hour := -1
	if d.byHour {
		hour = s.At.Hour()
	}
	st, b := stratum{s.Metric, s.Band, hour}, base{s.Metric, s.Band}
	if d.strata[st] == nil {
		d.strata[st] = &[outcomes]reservoir{}
	}
	if d.counts[b] == nil {
		d.counts[b] = &[outcomes]int{}
	}
	d.counts[b][s.Outcome]++
	d.strata[st][s.Outcome].keep(s, d.k, d.rng)
}

// set is the drawn set (B1), the strata in order: each as many closed as
// open, at most k, the closed half weak and half unheard as far as each
// goes and the rest from the other - so a constant answer scores 50 %.
func (d *drawer) set() ([]scenario, error) {
	if len(d.strata) == 0 {
		return nil, nil
	}
	keys := make([]stratum, 0, len(d.strata))
	for k := range d.strata {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b stratum) int {
		return int(a.Metric)*1e6 + int(a.Band)*100 + a.Hour - (int(b.Metric)*1e6 + int(b.Band)*100 + b.Hour)
	})
	var out []scenario
	for _, k := range keys {
		r := d.strata[k]
		o, w, u := d.shuffled(r[open].kept), d.shuffled(r[weak].kept), d.shuffled(r[unheard].kept)
		n := min(len(o), len(w)+len(u))
		nw := min(len(w), (n+1)/2)
		nu := min(len(u), n-nw)
		nw = min(len(w), n-nu)
		balanced := nw+nu == n && n <= d.k
		if err := invariant.Check(balanced, "a stratum was drawn with other than as many closed as open, or more than k: no figure from the draw stands"); err != nil {
			return nil, err
		}
		out = append(out, o[:n]...)
		out = append(out, w[:nw]...)
		out = append(out, u[:nu]...)
	}
	return out, nil
}

// shuffled is a copy of a sample in random order: a reservoir's slots are
// a uniform sample, not a random order.
func (d *drawer) shuffled(in []scenario) []scenario {
	if len(in) == 0 {
		return nil
	}
	if d.rng == nil {
		return slices.Clone(in)
	}
	out := slices.Clone(in)
	d.rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}
