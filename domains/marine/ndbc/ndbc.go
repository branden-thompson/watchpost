// Package ndbc reads NOAA National Data Buoy Center observations (B3 UAT 29,
// live-probed; free, no key): the active-station list gives every buoy's
// position, and each station's realtime2 text product carries the latest
// observed wave height (WVHT), dominant period (DPD), mean direction (MWD),
// and water temperature (WTMP) — the 5-day product. The nearest reporting buoy within the search
// radius supplies a location's Marine observation — Great Lakes buoys
// included, so inland lakes get coverage where buoys exist.
package ndbc

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Attribution is the NDBC data credit line.
const Attribution = "NOAA National Data Buoy Center (ndbc.noaa.gov)" // public domain

// MaxDistanceKM bounds the nearest-buoy search — beyond it a buoy no longer
// represents the location's waters.
const MaxDistanceKM = 150.0

// Provider is the NDBC snapshot provider.
type Provider struct {
	client *httpx.Client
	base   string

	mu       sync.Mutex
	stations []station
	loadedAt time.Time
}

// Cache lifetimes (UAT 71, stated per product — the client cache applies
// them): buoys report every 30 minutes, so a product is reused for 10; the
// coast-dense watchlist shares a handful of buoys. The station list turns
// over daily (NDBC declares 60 s on a 270 KB file).
const (
	obsTTL      = 10 * time.Minute
	stationsTTL = 24 * time.Hour
)

type station struct {
	ID   string  `xml:"id,attr"`
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Name string  `xml:"name,attr"`
	Met  string  `xml:"met,attr"`
	Type string  `xml:"type,attr"` // buoy | fixed (tide gauge / C-MAN) | oilrig | dart | ...
}

// New builds the provider. base "" means the production host.
func New(client *httpx.Client, base string) *Provider {
	if base == "" {
		base = "https://www.ndbc.noaa.gov"
	}
	return &Provider{client: client, base: base}
}

// ID implements snapshot.Provider.
func (p *Provider) ID() string { return "ndbc" }

// Domains implements snapshot.Provider: buoy files are observations — the
// fast marine tier (UAT 72).
func (p *Provider) Domains() []string { return []string{"marine-obs"} }

// Fetch implements snapshot.Provider for KindMarineObs.
func (p *Provider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	frag := snapshot.Fragment{Provider: p.ID(), Kind: req.Kind, FetchedAt: time.Now().UTC(), PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	if err := invariant.Check(req.Kind == snapshot.KindMarineObs, "ndbc serves only KindMarineObs"); err != nil {
		return frag, err
	}
	if err := p.loadStations(ctx); err != nil {
		frag.Err = err
		return frag, nil
	}
	// Concurrent + fail-soft per location (UAT 59/64): the rest of the batch
	// still lands, and every call reserves its pacing slot up front.
	got, err := snapshot.FetchEach(ctx, req.Locations, fetchConcurrency, func(ctx context.Context, ref snapshot.LocationRef) (snapshot.PartialData, error) {
		mar, err := p.marineFor(ctx, ref)
		if err != nil {
			return snapshot.PartialData{}, err
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

// fetchConcurrency bounds per-location fan-out inside one Fetch.
const fetchConcurrency = 6

// Fallback-chain depths (UAT 59 — San Francisco, downtown San Diego,
// Seattle, Miami): the nearest "active" station is usually a tide gauge
// with no wave product, so waves come from the nearest true buoys and water
// temperature from the nearest station of any kind that reports one (deep
// enough to reach a gauge across an inland sound — Seattle's is 38 km away,
// sixth by distance; products are cached, so the depth costs little).
const (
	waveCandidates = 4
	tempCandidates = 8
)

// marineFor resolves one location's sea state: the first buoy reporting
// waves is its buoy; water temperature is filled from the nearest station
// reporting one when that buoy lacks it. nil, nil when nothing in range
// represents these waters; an error only when every candidate failed.
func (p *Provider) marineFor(ctx context.Context, ref snapshot.LocationRef) (*snapshot.Marine, error) {
	near := p.nearest(ref.Lat, ref.Lon)
	var errs []error
	waves := p.firstReporting(ctx, buoysOnly(near, waveCandidates), func(m *snapshot.Marine) bool { return m.WaveHeight != nil }, &errs)
	if waves != nil && waves.WaterTemp != nil {
		return waves, nil
	}
	temp := p.firstReporting(ctx, head(near, tempCandidates), func(m *snapshot.Marine) bool { return m.WaterTemp != nil }, &errs)
	switch {
	case waves != nil:
		if temp != nil {
			waves.WaterTemp = keepOr(nil, temp.WaterTemp)
		}
		return waves, nil
	case temp != nil:
		return temp, nil
	case len(errs) > 0:
		return nil, fmt.Errorf("marine for %s: %w", ref.Label, errors.Join(errs...))
	}
	return nil, nil
}

// firstReporting walks candidates in order and returns a copy of the first
// product that satisfies ok; failures accumulate in errs and never abort.
func (p *Provider) firstReporting(ctx context.Context, cands []candidate, ok func(*snapshot.Marine) bool, errs *[]error) *snapshot.Marine {
	for _, c := range cands {
		mar, err := p.fetchStation(ctx, c.station, c.km)
		if err != nil {
			*errs = append(*errs, err)
			continue
		}
		if ok(mar) {
			return mar
		}
	}
	return nil
}

func buoysOnly(cands []candidate, n int) []candidate {
	var out []candidate
	for _, c := range cands {
		if c.station.Type == "buoy" && len(out) < n {
			out = append(out, c)
		}
	}
	return out
}

func head(cands []candidate, n int) []candidate {
	if len(cands) > n {
		return cands[:n]
	}
	return cands
}

// keepOr copies fallback when have is nil (P10-09 single-deref helper).
func keepOr(have, fallback *float64) *float64 {
	if have != nil || fallback == nil {
		return have
	}
	v := *fallback
	return &v
}

// loadStations fetches activestations.xml once a day.
func (p *Provider) loadStations(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.stations) > 0 && time.Since(p.loadedAt) < 24*time.Hour {
		return nil
	}
	raw, err := p.client.GetText(ctx, p.base+"/activestations.xml", httpx.TTL(stationsTTL))
	if err != nil {
		return fmt.Errorf("ndbc station list: %w", err)
	}
	var doc struct {
		Stations []station `xml:"station"`
	}
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("ndbc station list: %w", err)
	}
	kept := doc.Stations[:0]
	for _, s := range doc.Stations {
		if s.Met == "y" && s.ID != "" {
			kept = append(kept, s)
		}
	}
	if err := invariant.Check(len(kept) > 0, "ndbc: station list carried no reporting stations"); err != nil {
		return err
	}
	p.stations, p.loadedAt = kept, time.Now()
	return nil
}

// candidate is a station with its distance from the location.
type candidate struct {
	station station
	km      float64
}

// nearest returns every reporting station within MaxDistanceKM, nearest first.
func (p *Provider) nearest(lat, lon float64) []candidate {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []candidate
	for _, s := range p.stations {
		if d := geo.HaversineKM(lat, lon, s.Lat, s.Lon); d <= MaxDistanceKM {
			out = append(out, candidate{s, d})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].km < out[j].km })
	return out
}

// fetchStation parses a station's latest observation; the client cache
// serves nearby locations the same product for obsTTL.
// Product files are upper-case on the NDBC host even when the station list
// carries lower-case ids (tide gauges: "sdbc1" -> SDBC1_5day.txt).
func (p *Provider) fetchStation(ctx context.Context, st station, distKM float64) (*snapshot.Marine, error) {
	id := strings.ToUpper(st.ID)
	// 5day2 (UAT 72): the same columns as realtime2 at a ninth of the size
	// (23 KB vs 207 KB — 45 days of rows we never read).
	raw, err := p.client.GetText(ctx, fmt.Sprintf("%s/data/5day2/%s_5day.txt", p.base, id), httpx.TTL(obsTTL))
	if err != nil {
		return nil, fmt.Errorf("ndbc %s: %w", id, err)
	}
	mar, err := ParseRealtime(raw)
	if err != nil {
		return nil, fmt.Errorf("ndbc %s: %w", id, err)
	}
	mar.Buoy = id
	mar.BuoyDistanceKM = &distKM
	mar.Source = snapshot.SourceInfo{Provider: p.ID()}
	return mar, nil
}

// ParseRealtime reads an NDBC realtime2 standard-met text product: two
// header lines (names, units) then newest-first rows; "MM" marks missing.
func ParseRealtime(raw []byte) (*snapshot.Marine, error) {
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if err := invariant.Check(len(lines) >= 3 && strings.HasPrefix(lines[0], "#"), "ndbc: unrecognized realtime product"); err != nil {
		return nil, err
	}
	cols := strings.Fields(strings.TrimPrefix(lines[0], "#"))
	idx := map[string]int{}
	for i, c := range cols {
		idx[c] = i
	}
	row := strings.Fields(lines[2])
	if err := invariant.Check(len(row) == len(cols), "ndbc: observation row does not match its header"); err != nil {
		return nil, err
	}
	get := func(name string) *float64 {
		i, ok := idx[name]
		if !ok || row[i] == "MM" {
			return nil
		}
		v, err := strconv.ParseFloat(row[i], 64)
		if err != nil {
			return nil
		}
		return &v
	}
	mar := &snapshot.Marine{
		WaveHeight:  get("WVHT"),
		WavePeriod:  get("DPD"),
		SwellDirDeg: get("MWD"),
		WaterTemp:   get("WTMP"),
		WindSpeed:   get("WSPD"), // m/s (UAT 32.3)
		WindGust:    get("GST"),
	}
	if y, m, d, hh, mm := get("YY"), get("MM"), get("DD"), get("hh"), get("mm"); y != nil && m != nil && d != nil && hh != nil && mm != nil {
		mar.ObservedAt = time.Date(int(*y), time.Month(int(*m)), int(*d), int(*hh), int(*mm), 0, 0, time.UTC)
	}
	return mar, nil
}
