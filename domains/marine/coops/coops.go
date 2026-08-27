// Package coops reads NOAA CO-OPS Tides & Currents (B3 UAT 61, live-probed;
// free, no key): tide predictions (high/low, 3,499 stations), observed water
// level (301 gauges) and tidal-current predictions (4,430 stations). Each
// location takes the nearest station of each kind within its radius; the
// block merges into the Marine section beside the NWS forecast and NDBC
// buoy. Great Lakes gauges use lake datums, not MLLW — a follow-on.
package coops

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Attribution is the CO-OPS data credit line.
const Attribution = "NOAA CO-OPS Tides & Currents" // public domain (tidesandcurrents.noaa.gov)

// Radii: tides vary along a coast, currents with every headland — a
// prediction further away than these no longer describes the location.
const (
	TideRadiusKM    = 60.0
	CurrentRadiusKM = 40.0
)

// Cache lifetimes (UAT 71, stated per product because CO-OPS declares
// none — no-store on astronomical predictions): the client cache applies
// them, so a station's product is one download per lifetime however many
// locations share it.
const (
	stationsTTL    = 24 * time.Hour
	predictionsTTL = time.Hour        // the ceiling a caller TTL is never above; predictions live to UTC midnight (see untilUTCMidnight — Q5)
	levelTTL       = 10 * time.Minute // gauges report every 6 minutes
	rangeHours     = 48
)

// Provider is the CO-OPS snapshot provider.
type Provider struct {
	client *httpx.Client
	base   string
	Now    func() time.Time // clock (tests)

	mu       sync.Mutex
	tide     []station
	level    []station
	current  []station
	loadedAt time.Time
}

type station struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	Type string  `json:"type"` // current stations: H (harmonic) | S (subordinate) | W (weak & variable — no predictions)
}

// Fallback depths: a listed station can still answer "predictions are not
// available" (type-W current stations) or "No Predictions data was found …
// Datum" (tide stations without an MLLW datum — Philadelphia's, UAT 64);
// the chain walks to the next nearest.
const (
	tideCandidates    = 3
	currentCandidates = 3
)

// New builds the provider. base "" means the production host.
func New(client *httpx.Client, base string) *Provider {
	if base == "" {
		base = "https://api.tidesandcurrents.noaa.gov"
	}
	return &Provider{client: client, base: base, Now: time.Now}
}

// ID implements snapshot.Provider.
func (p *Provider) ID() string { return "coops" }

// CachedStations reports the memoised station lists' size (tide + level +
// current) for the diagnostic dump; they reload daily, never grow.
func (p *Provider) CachedStations() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.tide) + len(p.level) + len(p.current)
}

// Domains implements snapshot.Provider: predictions (tides, currents).
func (p *Provider) Domains() []string { return []string{"marine"} }

// Fetch implements snapshot.Provider for KindMarine (fail-soft per location).
func (p *Provider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	if err := invariant.Check(req.Kind == snapshot.KindMarine, "coops serves only KindMarine"); err != nil {
		return snapshot.Fragment{Provider: p.ID(), Kind: req.Kind}, err
	}
	return p.fetchEach(ctx, p.ID(), req, p.marineFor)
}

// fetchEach is the shared fan-out (UAT 64): every call reserves its pacing
// slot up front, so a two-location priority batch is never queued call-by-
// call behind the recent pipeline's fan-out.
func (p *Provider) fetchEach(ctx context.Context, id string, req snapshot.FetchReq,
	fn func(context.Context, snapshot.LocationRef) (*snapshot.Marine, error)) (snapshot.Fragment, error) {
	frag := snapshot.Fragment{Provider: id, Kind: req.Kind, FetchedAt: time.Now().UTC(), PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	if err := p.loadStations(ctx); err != nil {
		frag.Err = err
		return frag, nil
	}
	got, err := snapshot.FetchEach(ctx, req.Locations, fetchConcurrency, func(ctx context.Context, ref snapshot.LocationRef) (snapshot.PartialData, error) {
		mar, err := fn(ctx, ref)
		if err != nil {
			return snapshot.PartialData{}, fmt.Errorf("tides for %s: %w", ref.Label, err)
		}
		return snapshot.PartialData{Marine: mar}, nil
	})
	for k, pd := range got {
		if pd.Marine != nil {
			frag.PerLocation[k] = pd
		}
	}
	frag.Err = err
	return frag, nil
}

// ObsProvider is the observation half of CO-OPS (UAT 72): the water level
// at the nearest gauge, on the fast marine-obs tier. It shares the station
// lists and client with the predictions provider.
type ObsProvider struct{ p *Provider }

// NewObs wraps a predictions provider.
func NewObs(p *Provider) *ObsProvider { return &ObsProvider{p: p} }

// ID implements snapshot.Provider.
func (o *ObsProvider) ID() string { return "coops-obs" }

// Domains implements snapshot.Provider.
func (o *ObsProvider) Domains() []string { return []string{"marine-obs"} }

// Fetch implements snapshot.Provider for KindMarineObs.
func (o *ObsProvider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	if err := invariant.Check(req.Kind == snapshot.KindMarineObs, "coops-obs serves only KindMarineObs"); err != nil {
		return snapshot.Fragment{Provider: o.ID(), Kind: req.Kind}, err
	}
	return o.p.fetchEach(ctx, o.ID(), req, o.p.levelFor)
}

// levelFor reads the observed water level at the nearest gauge in range;
// nil, nil when none or when the gauge answers an error envelope (a gauge
// outage never fails the batch — the predictions provider owns the section).
func (p *Provider) levelFor(ctx context.Context, ref snapshot.LocationRef) (*snapshot.Marine, error) {
	ls, _, ok := nearest(p.level, ref.Lat, ref.Lon, TideRadiusKM)
	if !ok {
		return nil, nil
	}
	lvl, err := p.fetchLevel(ctx, ls.ID)
	if err != nil || lvl == nil {
		return nil, nil
	}
	return &snapshot.Marine{TideLevel: lvl, Source: snapshot.SourceInfo{Provider: "coops-obs"}}, nil
}

// fetchConcurrency bounds per-location fan-out inside one Fetch.
const fetchConcurrency = 6

// marineFor assembles one location's tide block: predictions from the
// nearest tide station (required — no station in range means no block),
// the observed level from the nearest gauge, and currents from the nearest
// current station, each within its radius.
func (p *Provider) marineFor(ctx context.Context, ref snapshot.LocationRef) (*snapshot.Marine, error) {
	cands := nearestN(p.tide, ref.Lat, ref.Lon, TideRadiusKM, tideCandidates)
	if len(cands) == 0 {
		return nil, nil
	}
	mar := &snapshot.Marine{Source: snapshot.SourceInfo{Provider: p.ID()}}
	// Predictions fetch concurrently (UAT 64); currents are a nicety. The
	// observed level lives on ObsProvider (UAT 72: the fast tier).
	var g errgroup.Group
	var tidesErr error
	g.Go(func() error {
		mar.Tides, mar.TideStation, mar.TideStationKM, tidesErr = p.tidesFor(ctx, cands)
		return nil
	})
	g.Go(func() error {
		mar.Currents, mar.CurrentStation = p.currentsFor(ctx, ref)
		return nil
	})
	_ = g.Wait()
	if tidesErr != nil {
		return nil, tidesErr
	}
	return mar, nil
}

// tidesFor walks the nearest tide stations until one answers with
// predictions; the last error surfaces only when every candidate failed.
func (p *Provider) tidesFor(ctx context.Context, cands []candidate) ([]snapshot.TideEvent, string, *float64, error) {
	var lastErr error
	for _, c := range cands {
		st, km := c.station, c.km
		tides, err := p.fetchTides(ctx, st.ID)
		if err != nil {
			lastErr = err
			continue
		}
		return tides, st.Name, &km, nil
	}
	return nil, "", nil, lastErr
}

// currentsFor walks the nearest predictable current stations.
func (p *Provider) currentsFor(ctx context.Context, ref snapshot.LocationRef) ([]snapshot.CurrentEvent, string) {
	for _, cs := range nearestN(p.current, ref.Lat, ref.Lon, CurrentRadiusKM, currentCandidates) {
		st := cs.station
		if cur, err := p.fetchCurrents(ctx, st.ID); err == nil && len(cur) > 0 {
			return cur, st.Name
		}
	}
	return nil, ""
}

// predictable drops type-W current stations: CO-OPS lists them but answers
// "predictions are not available" (downtown San Diego's B St. Pier).
func predictable(list []station) []station {
	kept := list[:0]
	for _, s := range list {
		if s.Type != "W" {
			kept = append(kept, s)
		}
	}
	return kept
}

// loadStations fetches the three station lists once a day.
func (p *Provider) loadStations(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.tide) > 0 && time.Since(p.loadedAt) < stationsTTL {
		return nil
	}
	lists := []struct {
		kind string
		dst  *[]station
	}{{"tidepredictions", &p.tide}, {"waterlevels", &p.level}, {"currentpredictions", &p.current}}
	for _, l := range lists {
		var doc struct {
			Stations []station `json:"stations"`
		}
		if _, err := p.client.GetJSON(ctx, p.base+"/mdapi/prod/webapi/stations.json?type="+l.kind, &doc, httpx.TTL(stationsTTL)); err != nil {
			return fmt.Errorf("coops %s stations: %w", l.kind, err)
		}
		*l.dst = doc.Stations
	}
	p.current = predictable(p.current)
	if err := invariant.Check(len(p.tide) > 0, "coops: tide station list is empty"); err != nil {
		return err
	}
	p.loadedAt = time.Now()
	return nil
}

// nearest returns the closest station within radiusKM.
func nearest(list []station, lat, lon, radiusKM float64) (station, float64, bool) {
	if c := nearestN(list, lat, lon, radiusKM, 1); len(c) > 0 {
		return c[0].station, c[0].km, true
	}
	return station{}, 0, false
}

type candidate struct {
	station station
	km      float64
}

// nearestN returns up to n stations within radiusKM, nearest first.
func nearestN(list []station, lat, lon, radiusKM float64, n int) []candidate {
	var out []candidate
	for _, s := range list {
		if d := geo.HaversineKM(lat, lon, s.Lat, s.Lng); d <= radiusKM {
			out = append(out, candidate{s, d})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].km < out[j].km })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// query builds a datagetter URL: GMT, metric, JSON.
func (p *Provider) query(product, stationID string, extra url.Values) string {
	q := url.Values{"product": {product}, "station": {stationID}, "time_zone": {"gmt"}, "units": {"metric"}, "format": {"json"}}
	for k, v := range extra {
		q[k] = v
	}
	return p.base + "/api/prod/datagetter?" + q.Encode()
}

// window is the prediction window: from today 00:00 UTC for rangeHours.
func (p *Provider) window() url.Values {
	return url.Values{"begin_date": {p.Now().UTC().Format("20060102")}, "range": {strconv.Itoa(rangeHours)}}
}

// apiError reads the HTTP-200 error envelope CO-OPS uses for bad requests.
type apiError struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e apiError) err() error {
	if e.Error != nil {
		return fmt.Errorf("coops: %s", e.Error.Message)
	}
	return nil
}

func (p *Provider) fetchTides(ctx context.Context, id string) ([]snapshot.TideEvent, error) {
	var doc struct {
		apiError
		Predictions []struct {
			T, V, Type string
		} `json:"predictions"`
	}
	extra := p.window()
	extra.Set("datum", "MLLW")
	extra.Set("interval", "hilo")
	if _, err := p.client.GetJSON(ctx, p.query("predictions", id, extra), &doc, httpx.TTL(p.predictionsLifetime())); err != nil {
		return nil, err
	}
	if err := doc.err(); err != nil {
		return nil, err
	}
	out := make([]snapshot.TideEvent, 0, len(doc.Predictions))
	for _, pr := range doc.Predictions {
		t, v, ok := parseTV(pr.T, pr.V)
		if ok && (pr.Type == "H" || pr.Type == "L") {
			out = append(out, snapshot.TideEvent{Time: t, Height: v, Type: pr.Type})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, nil
}

func (p *Provider) fetchLevel(ctx context.Context, id string) (*float64, error) {
	var doc struct {
		apiError
		Data []struct{ T, V string } `json:"data"`
	}
	if _, err := p.client.GetJSON(ctx, p.query("water_level", id, url.Values{"datum": {"MLLW"}, "date": {"latest"}}), &doc, httpx.TTL(levelTTL)); err != nil {
		return nil, err
	}
	if err := doc.err(); err != nil {
		return nil, err
	}
	if len(doc.Data) == 0 {
		return nil, nil
	}
	if _, v, ok := parseTV(doc.Data[0].T, doc.Data[0].V); ok {
		return &v, nil
	}
	return nil, nil
}

func (p *Provider) fetchCurrents(ctx context.Context, id string) ([]snapshot.CurrentEvent, error) {
	var doc struct {
		apiError
		CP struct {
			Events []struct {
				Type          string  `json:"Type"`
				Time          string  `json:"Time"`
				VelocityMajor float64 `json:"Velocity_Major"` // cm/s, ebb negative
			} `json:"cp"`
		} `json:"current_predictions"`
	}
	extra := p.window()
	extra.Set("interval", "MAX_SLACK")
	if _, err := p.client.GetJSON(ctx, p.query("currents_predictions", id, extra), &doc, httpx.TTL(p.predictionsLifetime())); err != nil {
		return nil, err
	}
	if err := doc.err(); err != nil {
		return nil, err
	}
	out := make([]snapshot.CurrentEvent, 0, len(doc.CP.Events))
	for _, e := range doc.CP.Events {
		t, err := parseTime(e.Time)
		if err != nil {
			continue
		}
		out = append(out, snapshot.CurrentEvent{Time: t, Speed: math.Abs(e.VelocityMajor) / 100, Type: e.Type})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, nil
}

// parseTime reads the API's "2006-01-02 15:04" GMT stamps.
func parseTime(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04", s, time.UTC)
}

// parseTV reads a (time, value) pair; blank values (gauge gaps) are skipped.
func parseTV(t, v string) (time.Time, float64, bool) {
	ts, err := parseTime(t)
	if err != nil {
		return time.Time{}, 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return time.Time{}, 0, false
	}
	return ts, f, true
}

// predictionsLifetime is how long a predictions answer is cached (quality
// pass Q5, plan Q5b-5): the window is keyed by today's UTC date, so the
// answer is the same until UTC midnight and the URL itself changes after
// it — one fetch per station per day instead of one per hour. Never past
// UTC midnight; never less than a minute (a request just before midnight
// still caches briefly).
func (p *Provider) predictionsLifetime() time.Duration { return untilUTCMidnight(p.Now()) }

func untilUTCMidnight(now time.Time) time.Duration {
	u := now.UTC()
	midnight := time.Date(u.Year(), u.Month(), u.Day()+1, 0, 0, 0, 0, time.UTC)
	return max(time.Minute, midnight.Sub(u))
}
