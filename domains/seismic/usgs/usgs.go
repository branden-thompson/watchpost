// Package usgs reads the USGS Earthquake Hazards Program FDSN event feed
// (0.11.0, live-probed 2026-08-27; keyless, public domain). Because the
// graduated rule is concentric — a small quake shows only if very close, a
// large one from far away — the fetch is concentric too (seismic P2): a tight
// near-field query (low magnitude, small radius, centred on the location) and
// a wide regional query (≥ the pivot magnitude) snapped to a shared grid so
// nearby locations collapse onto one request. Unioned and Keep-filtered, the
// result is identical to one wide magnitude-0 query — which would pull ~1 MB
// of low-magnitude events the rule then discards (measured 2026-08-27). USGS
// gives neither distance nor bearing from the location — those are computed
// here.
package usgs

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/branden-thompson/watchpost/domains/seismic"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Attribution is the credit line (public domain). Kept ≤ 52 cells to fit the
// About window (app credits_test); the comma form drops one cell vs parens.
const Attribution = "USGS Earthquake Hazards Program, earthquake.usgs.gov"

// feedTTL is the caller cache lifetime. USGS cannot answer a conditional GET
// (it stamps Last-Modified and the body's metadata.generated with the
// per-request time, so If-Modified-Since always draws a 200, never a 304 —
// measured at the P2 gate), so the byte floor is set by refetch frequency,
// not revalidation. Ten minutes — above the 5-min priority cadence — lets a
// favourite refetch only every other tick and lets the RECENT tier and
// shared regional boxes ride those refreshes at zero bytes (HUM LEAD, P2
// gate, decision A). A quake is no more current than the feed; "did my area
// shake" tolerates minutes of staleness (plan §0.3).
const feedTTL = 10 * time.Minute

// maxQuakes caps a location's list (P10-03: a swarm cannot grow the section
// or the memo without bound).
const maxQuakes = 20

// Plausibility bounds reject malformed or hostile feed values at decode
// (REVIEW P5 E-A/E-C): real quakes span roughly magnitude −2 to 10 at 0 to
// 1000 km depth (the deepest recorded is ~700 km), and no real 7-day box holds
// anywhere near maxFeatures events — the cap is a pure hostile-payload backstop
// so the parsed-box memo entry cannot grow without bound.
const (
	minPlausibleMag     = -2.0
	maxPlausibleMag     = 10.0
	maxPlausibleDepthKm = 1000.0
	maxFeatures         = 20000
)

// regionalMinMi: a query wider than this snaps to the shared grid; a tighter
// (near-field) query stays per-location. Snapping a near query would inflate
// its radius by the cell's buffer, and low-magnitude counts scale with area —
// in an active swarm that is a large, needless payload (measured 2026-08-27:
// a 0.5° near snap took Ridgecrest from 28 KB to 254 KB). The regional query
// is sparse (M ≥ pivot) and swarm-insensitive, so its buffer is free.
const regionalMinMi = 200

// regPitchDeg is the regional grid pitch (degrees): every location in a ~4°
// cell shares one regional request — the FIRMS-tile precedent (Q5). The query
// radius adds the cell's half-diagonal so the snapped circle still covers each
// location's own reach; Keep then trims to the exact per-location set (the
// equivalence property).
const regPitchDeg = 4.0

// Provider is the USGS seismic provider.
type Provider struct {
	client *httpx.Client
	base   string
	rules  seismic.Rules
	memo   *boxMemo
}

// New builds the provider; base "" means the production FDSN endpoint.
func New(client *httpx.Client, base string, rules seismic.Rules) *Provider {
	if base == "" {
		base = "https://earthquake.usgs.gov/fdsnws/event/1/query"
	}
	return &Provider{client: client, base: base, rules: rules, memo: newBoxMemo()}
}

// ID implements snapshot.Provider.
func (p *Provider) ID() string { return "usgs" }

// Domains implements snapshot.Provider.
func (p *Provider) Domains() []string { return []string{"seismic"} }

// MemoStats reports the parsed-box memo's size and parses since launch (the
// [S] gauge; bounded by maxBoxes).
func (p *Provider) MemoStats() (boxes, parses int) { return p.memo.stats() }

// boxQuery is one concentric request: its URL (the cache + singleflight key)
// and the magnitude window it is responsible for (guarded locally so the
// near/regional overlap at the pivot cannot double-count).
type boxQuery struct {
	url            string
	minMag, maxMag float64 // maxMag 0 ⇒ open
}

// queries builds the concentric requests for one location: the near-field
// query centred on the location itself (no snap — the buffer would balloon a
// swarm), and the regional query snapped to the shared grid so nearby
// locations resolve to one URL.
func (p *Provider) queries(ref snapshot.LocationRef, now time.Time) []boxQuery {
	plan := p.rules.QueryPlan()
	out := make([]boxQuery, 0, len(plan))
	for _, bq := range plan {
		lat, lon, radiusKm := ref.Lat, ref.Lon, bq.RadiusMi*seismic.MileKm
		if bq.RadiusMi >= regionalMinMi {
			lat, lon = snapDeg(ref.Lat, regPitchDeg), snapDeg(ref.Lon, regPitchDeg)
			radiusKm += halfDiagKm(regPitchDeg)
		}
		out = append(out, boxQuery{url: p.queryURL(lat, lon, radiusKm, bq.MinMag, bq.MaxMag, now), minMag: bq.MinMag, maxMag: bq.MaxMag})
	}
	return out
}

// snapDeg rounds a coordinate to the grid so nearby centres share a URL.
func snapDeg(v, pitch float64) float64 { return math.Round(v/pitch) * pitch }

// halfDiagKm is half a square grid cell's diagonal, in km, taken at the
// equator (the widest longitude) so the buffer is safe at every latitude.
func halfDiagKm(pitchDeg float64) float64 { return math.Sqrt2 * pitchDeg * 111.32 / 2 }

// queryURL builds one FDSN circle query. starttime is the lookback window;
// maxmagnitude is set only for a bounded (near-field) window.
func (p *Provider) queryURL(lat, lon, radiusKm, minMag, maxMag float64, now time.Time) string {
	q := url.Values{}
	q.Set("format", "geojson")
	q.Set("latitude", fmt.Sprintf("%.4f", lat))
	q.Set("longitude", fmt.Sprintf("%.4f", lon))
	q.Set("maxradiuskm", fmt.Sprintf("%.0f", math.Ceil(radiusKm))) // round UP: truncating (e.g. 32.187→"32") would drop a Keep-visible quake in the last fraction of a mile (REVIEW P5 F1)
	q.Set("minmagnitude", fmt.Sprintf("%.2f", minMag))
	if maxMag != 0 {
		q.Set("maxmagnitude", fmt.Sprintf("%.2f", maxMag))
	}
	q.Set("starttime", now.UTC().AddDate(0, 0, -p.rules.LookbackDays).Format("2006-01-02"))
	q.Set("orderby", "time")
	return p.base + "?" + q.Encode()
}

// feature is one decoded USGS event (the memo's unit).
type feature struct {
	id       string
	mag      float64
	magType  string
	place    string
	at       time.Time
	depthKm  float64
	lat, lon float64
	tsunami  bool
	alert    string
	felt     *int
	sig      int
	typ      string
}

// geoJSON is the slice of the USGS FeatureCollection the section reads.
type geoJSON struct {
	Features []struct {
		ID         string `json:"id"`
		Properties struct {
			Mag     *float64 `json:"mag"`
			MagType string   `json:"magType"`
			Place   string   `json:"place"`
			Time    int64    `json:"time"` // epoch ms
			Tsunami int      `json:"tsunami"`
			Alert   string   `json:"alert"`
			Felt    *int     `json:"felt"`
			Sig     int      `json:"sig"`
			Type    string   `json:"type"`
		} `json:"properties"`
		Geometry struct {
			Coordinates []float64 `json:"coordinates"` // [lon, lat, depthKm]
		} `json:"geometry"`
	} `json:"features"`
}

// parseFeatures decodes a box body into features, dropping the malformed
// (no magnitude, no position) — the rule and distance are applied later.
func parseFeatures(raw []byte) ([]feature, error) {
	var gj geoJSON
	if err := json.Unmarshal(raw, &gj); err != nil {
		return nil, err
	}
	out := make([]feature, 0, min(len(gj.Features), maxFeatures))
	for _, f := range gj.Features {
		if len(out) >= maxFeatures { // a malformed/hostile feed cannot make the memo entry unbounded (REVIEW P5 E-C)
			break
		}
		g := f.Geometry.Coordinates
		// Drop the malformed AND the implausible: a real quake is roughly
		// magnitude −2..10 at 0..1000 km depth. An out-of-range value (a
		// hostile mag=1e300) would sort first and render/speak a ~300-character
		// token that tears the modal or bloats the broadcast (REVIEW P5 E-A).
		if f.Properties.Mag == nil || len(g) < 2 {
			continue
		}
		mag := *f.Properties.Mag
		if mag < minPlausibleMag || mag > maxPlausibleMag {
			continue
		}
		depth := 0.0
		if len(g) >= 3 {
			depth = g[2]
		}
		if depth < 0 || depth > maxPlausibleDepthKm {
			continue
		}
		out = append(out, feature{
			id: f.ID, mag: mag, magType: f.Properties.MagType, place: f.Properties.Place,
			at: time.UnixMilli(f.Properties.Time).UTC(), depthKm: depth, lat: g[1], lon: g[0],
			tsunami: f.Properties.Tsunami != 0, alert: f.Properties.Alert, felt: f.Properties.Felt,
			sig: f.Properties.Sig, typ: f.Properties.Type,
		})
	}
	return out, nil
}

// Fetch implements snapshot.Provider for KindSeismic: every requested location
// gets a SeismicState — its recent quakes within the graduated bands,
// largest-then-nearest. The concentric queries are unioned (deduped by event
// id) and then Keep-filtered, so the result equals one wide query. AsOf marks
// a feed that answered (even with no quakes) apart from one that did not
// (zero — never "no quakes").
func (p *Provider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	frag := snapshot.Fragment{Provider: p.ID(), Kind: req.Kind, FetchedAt: time.Now().UTC(), PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	if err := invariant.Check(req.Kind == snapshot.KindSeismic, "usgs serves only KindSeismic"); err != nil {
		return frag, err
	}
	if err := p.rules.Valid(); err != nil {
		return frag, err
	}
	now := time.Now()
	for _, ref := range req.Locations {
		feats, ok, err := p.gather(ctx, ref, now)
		if !ok {
			// Any failed leg leaves the location's prior state untouched (the
			// scheduler retries). Publishing a partial answer would replace a
			// complete last-good state with one leg's subset and silently drop
			// the other leg's events — e.g. a known distant M5 vanishing when
			// only the regional query blips (REVIEW P5 F1).
			frag.Err = fmt.Errorf("usgs: %w", err)
			continue
		}
		frag.PerLocation[snapshot.Key(ref)] = snapshot.PartialData{Seismic: p.stateFor(ref, feats, frag.FetchedAt)}
	}
	return frag, nil
}

// gather runs a location's concentric queries and returns the merged,
// deduped features. ok is true only when EVERY query answered — a partial
// answer is incomplete (it is missing one leg's whole magnitude band), so the
// caller keeps last-good rather than publishing a subset (REVIEW P5 F1). err
// carries the first failure.
func (p *Provider) gather(ctx context.Context, ref snapshot.LocationRef, now time.Time) (feats []feature, ok bool, err error) {
	ok = true
	seen := map[string]bool{}
	for _, q := range p.queries(ref, now) {
		raw, gerr := p.client.GetText(ctx, q.url, httpx.TTL(feedTTL))
		if gerr != nil {
			ok, err = false, gerr
			continue
		}
		parsed, perr := p.memo.features(q.url, raw)
		if perr != nil {
			ok, err = false, perr
			continue
		}
		for _, f := range parsed {
			// The window is enforced server-side; the local guard makes it
			// half-open [min,max) so the pivot-magnitude overlap between the
			// near and regional queries cannot double-show one event.
			if f.mag < q.minMag || (q.maxMag != 0 && f.mag >= q.maxMag) {
				continue
			}
			key := f.id
			if key == "" { // fixtures may omit the id; fall back to a natural key
				key = fmt.Sprintf("%.4f,%.4f,%d,%.2f", f.lat, f.lon, f.at.UnixMilli(), f.mag)
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			feats = append(feats, f)
		}
	}
	return feats, ok, err
}

// stateFor filters a location's merged features by the graduated rule and the
// type allowlist, computes distance/bearing from the location, sorts
// largest-then-nearest and caps.
func (p *Provider) stateFor(ref snapshot.LocationRef, feats []feature, asOf time.Time) *snapshot.SeismicState {
	quakes := make([]snapshot.Quake, 0, len(feats))
	for _, f := range feats {
		if !p.rules.Wants(f.typ) {
			continue
		}
		km := geo.HaversineKM(ref.Lat, ref.Lon, f.lat, f.lon)
		if !p.rules.Keep(f.mag, km) {
			continue
		}
		quakes = append(quakes, snapshot.Quake{
			Mag: f.mag, MagType: f.magType, Place: f.place, DepthKm: f.depthKm,
			At: f.at, DistanceKm: km, Bearing: bearing(geo.BearingDeg(ref.Lat, ref.Lon, f.lat, f.lon)),
			Tsunami: f.tsunami, Alert: f.alert, Felt: f.felt, Sig: f.sig,
			Source: snapshot.SourceInfo{Provider: p.ID(), IssuedAt: f.at},
		})
	}
	seismic.Sort(quakes)
	if len(quakes) > maxQuakes {
		quakes = quakes[:maxQuakes]
	}
	return &snapshot.SeismicState{AsOf: asOf, Quakes: quakes}
}

// bearing names a direction on the 16-point compass (the consumers keep their
// own word tables — Q6 L3-F9; the arithmetic is geo.CompassIndex). The table is
// a local per the codebase's compass-table convention (P10-06); this runs once
// per quake per fetch (cold path, 10-min TTL).
func bearing(deg float64) string {
	pts := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	return pts[geo.CompassIndex(deg, 16)]
}
