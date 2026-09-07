package app

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// The CARD-BUILD instrument (07-readiness/perf-protocol.md §6 owns the method;
// this file only implements it). It measures the input to PD-2 — whether the
// Director should PRE-BUILD the next main-track card while the current one
// plays — by timing what stands between "the previous report ended" and "the
// next report can speak its first word".
//
// What it is NOT measuring, and why. Audio look-ahead inside a report already
// exists: synth.Source renders one segment ahead of playback on its own
// goroutine and the writer is forbidden to synthesise at all
// (domains/radio/synth/source.go — Open's one-deep `rendered` channel, and the
// writerKey seam that lets a test assert zero synthesiser calls on the writer).
// So there is no gap BETWEEN segments. The gap is at the CARD boundary, and it
// is data assembly rather than speech: segments() issues its fetches in a
// straight line — observation, alerts, office, products, forecast zone, county
// — with no concurrency, after tune() has already done a county lookup and a
// relay resolve. All of it runs after the previous report has gone quiet.
//
// LIVE BY DESIGN, and opt-in. A recorded fixture would measure the fixture: the
// number that decides PD-2 is how long real sequential calls to a real weather
// service take. Set WATCHPOST_LIVE_PERF=1 to run it; `make verify` never does.

// bonsall is the application's default location (R-4), so the measurement is
// taken where the app's own default listener stands.
func bonsall() snapshot.LocationRef {
	return snapshot.LocationRef{Label: "Bonsall, CA 92003", Zip: "92003", Lat: 33.2870, Lon: -117.2250, TZ: "America/Los_Angeles"}
}

// cardBuildClient builds the client THE WAY THE APP DOES — `RatePerSec: 30`,
// as `app/app.go:123` sets it.
//
// It is a named constructor rather than an inline literal because leaving the
// rate unset was a real defect in the first version of this instrument: the
// zero value takes httpx's default of 5/sec (`httpx.go:232`), and eleven
// requests at 5/sec is ~2.2 s — so the first measurement was reporting the token
// bucket rather than the network, at one sixth the app's true rate, and it
// looked like a plausible network figure. An instrument that measures its own
// configuration mistake reads exactly like a measurement.
func cardBuildClient(t testing.TB) *httpx.Client {
	t.Helper()
	client, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: 30, MaxRetries: 1, CacheDir: t.TempDir()})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return client
}

// cardBuildDeck is the smallest deck segments() needs. resolver is nil, which
// stationFor handles: the transmitter the lead names is not on the measured
// path, and standing one up would add a resolve this instrument does not claim
// to measure.
func cardBuildDeck(t testing.TB, client *httpx.Client) *radioDeck {
	t.Helper()
	return &radioDeck{
		nws:      nws.New(client, ""),
		products: synth.NewProducts(client, ""),
		units:    render.UnitF,
	}
}

// TestCardBuildCost reports what one main-track card costs to assemble, cold
// and then warm, in wall-clock and in network requests.
//
// Cold is the number that matters: it is what a listener waits through at every
// watchlist advance today, and what moving the build to standby would remove.
// Warm says how much of it the httpx cache already absorbs on a repeat, which
// bounds how much a pre-build could ever save on a location recently read.
func TestCardBuildCost(t *testing.T) {
	if os.Getenv("WATCHPOST_LIVE_PERF") != "1" {
		t.Skip("live measurement: set WATCHPOST_LIVE_PERF=1 (perf-protocol.md §6)")
	}
	client := cardBuildClient(t)
	d := cardBuildDeck(t, client)
	ref := bonsall()

	cold := measureCardBuild(t, d, client, ref)
	warm := measureCardBuild(t, d, client, ref)

	// THE INSTRUMENT VALIDATES ITSELF BEFORE IT REPORTS. A build that produced
	// no segments, or that never reached the network on the cold pass, has
	// measured nothing — and a zero from a broken instrument reads exactly like
	// a fast path, which is how four false numbers reached commit messages this
	// release (remediation-review-loop.md).
	if cold.segments == 0 {
		t.Fatal("the cold build produced no segments — the instrument is measuring nothing")
	}
	if cold.net == 0 {
		t.Fatal("the cold build made no network request — the cache was already warm, so this is not a cold build")
	}

	t.Logf("card build, %s", ref.Label)
	t.Logf("  COLD  %8s  net=%d cache=%d  segments=%d", cold.elapsed.Round(time.Millisecond), cold.net, cold.cache, cold.segments)
	t.Logf("  WARM  %8s  net=%d cache=%d  segments=%d", warm.elapsed.Round(time.Millisecond), warm.net, warm.cache, warm.segments)
	t.Logf("  the cold figure is the silence a watchlist advance costs today, before the first word")
}

// TestCardBuildConcurrentFloor measures what CONCURRENCY alone would buy, and
// what it would cost in extra requests (PL-3).
//
// The plan proposed to hide the card build behind standby. Before building a
// standby lifecycle it is worth knowing whether the build can simply be made
// fast: `segments()` issues its fetches in a straight line, but only one pair is
// genuinely dependent — `products` needs `office`. The rest are independent.
//
// It measures the FLOOR, not a proposed implementation: the same calls, issued
// concurrently, with the dependent one after its input. If the floor sits under
// the ≈0.7 s output buffer (perf-protocol §1), standby is solving a problem that
// concurrency removes outright.
//
// It also reports the REQUEST COUNT, because concurrency has a plausible cost
// here: several of these resolve the same `/points` lookup, which the sequential
// form gets from cache after the first. Fired together they can all miss — a
// cache stampede that trades wall clock for load on a public weather service.
func TestCardBuildConcurrentFloor(t *testing.T) {
	if os.Getenv("WATCHPOST_LIVE_PERF") != "1" {
		t.Skip("live measurement: set WATCHPOST_LIVE_PERF=1 (perf-protocol.md §6)")
	}
	client := cardBuildClient(t)
	d := cardBuildDeck(t, client)
	ref := bonsall()
	ctx := context.Background()

	before := totalRequests(client)
	start := time.Now()

	var wg sync.WaitGroup
	var office, zone, county string
	var obsErr, alertErr error
	for _, kind := range []snapshot.FetchKind{snapshot.KindObs, snapshot.KindAlerts} {
		wg.Add(1)
		go func(k snapshot.FetchKind) {
			defer wg.Done()
			_, err := d.nws.Fetch(ctx, snapshot.FetchReq{Kind: k, Locations: []snapshot.LocationRef{ref}})
			if k == snapshot.KindObs {
				obsErr = err
			} else {
				alertErr = err
			}
		}(kind)
	}
	wg.Add(3)
	go func() { defer wg.Done(); office = d.nws.Office(ctx, ref) }()
	go func() { defer wg.Done(); zone = d.nws.ForecastZone(ctx, ref) }()
	go func() { defer wg.Done(); county = d.nws.CountyUGC(ctx, ref) }()
	wg.Wait()

	// The ONE genuine dependency: the products query needs the office.
	products, _ := d.products.Latest(ctx, office)
	elapsed := time.Since(start)
	after := totalRequests(client)

	// SELF-VALIDATION. A concurrent run that resolved nothing would report a
	// beautiful number for doing no work — the same shape as the rate-limiter
	// mistake this file already made once.
	if office == "" || zone == "" || county == "" {
		t.Fatalf("the concurrent build resolved nothing (office=%q zone=%q county=%q) — measuring nothing", office, zone, county)
	}
	if obsErr != nil && alertErr != nil {
		t.Fatalf("both fetches failed (%v / %v) — measuring nothing", obsErr, alertErr)
	}
	if after.net-before.net == 0 {
		t.Fatal("no network request — the cache was warm, so this is not a cold build")
	}

	t.Logf("card build FLOOR (concurrent), %s", ref.Label)
	t.Logf("  %8s  net=%d cache=%d  products=%d", elapsed.Round(time.Millisecond), after.net-before.net, after.cache-before.cache, len(products))
	t.Logf("  compare with the sequential cold figure from TestCardBuildCost")
	t.Logf("  a request count ABOVE the sequential one is a cache stampede — the cost of the speed-up")
}

type cardBuild struct {
	elapsed    time.Duration
	net, cache int64
	segments   int
}

// measureCardBuild times one segments() call and attributes its requests.
func measureCardBuild(t testing.TB, d *radioDeck, client *httpx.Client, ref snapshot.LocationRef) cardBuild {
	t.Helper()
	before := totalRequests(client)
	start := time.Now()
	segs, err := d.segments(context.Background(), ref, synth.VoiceToken)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("segments: %v", err)
	}
	after := totalRequests(client)
	return cardBuild{elapsed: elapsed, net: after.net - before.net, cache: after.cache - before.cache, segments: len(segs)}
}

type reqTotals struct{ net, cache int64 }

// totalRequests sums every host's counters, because the card build spans
// several services and the question is how many round-trips it costs in all.
func totalRequests(client *httpx.Client) reqTotals {
	var out reqTotals
	for _, h := range client.RequestStats().Hosts {
		out.net += h.Net
		out.cache += h.Cache
	}
	return out
}
