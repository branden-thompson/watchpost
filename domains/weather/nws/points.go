package nws

// points.go — the /points resolution memo: grid, forecast and observation-station chain per location, the preferred-station mark. Split from provider.go by the quality pass (Q2, pure move).

import (
	"context"
	"fmt"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// obsStation is one observation station with its position (UAT 60: the
// table shows the station and its distance from the location).
type obsStation struct {
	id       string
	lat, lon float64
}

type gridInfo struct {
	forecastURL string
	hourlyURL   string
	gridURL     string       // raw gridpoint: marine swell/wave fields for coastal grids (UAT 29)
	stationsURL string       //
	stations    []obsStation // nearest-first observation stations (fallback chain)
	preferred   string       // station that last reported completely (guarded by Provider.mu)
	zones       []string     // UGC codes: forecastZone + county
	fireZone    string       // fire weather zone UGC (D-22 setup inference)
	timeZone    string
	resolvedAt  time.Time // when /points answered: a resolution older than gridTTL is redone (Q5, L4-F7 — NWS re-grids a point now and then)
}

// gridTTL is how long a point's resolution is trusted before /points is
// asked again (Q5b-7): a day — grids and station lists change rarely, and
// the cache serves the request when the answer has not.
const gridTTL = 24 * time.Hour

func (p *Provider) resolve(ctx context.Context, ref snapshot.LocationRef) (*gridInfo, error) {
	k := snapshot.Key(ref)
	p.mu.Lock()
	cached, ok := p.cache[k]
	p.mu.Unlock()
	if ok && p.now().Sub(cached.resolvedAt) < gridTTL {
		return cached, nil
	}
	// Singleflight (UAT 59): the alerts/obs/forecast tiers all resolve the
	// same location at launch — one points+stations round trip serves them.
	v, err, _ := p.sf.Do(string(k), func() (any, error) {
		g, err := p.resolvePoints(ctx, ref)
		if err != nil {
			if ok {
				return cached, nil // a stale resolution outlives a failed refresh: last-good, retried next time
			}
			return nil, err
		}
		g.resolvedAt = p.now()
		if ok {
			g.preferred = cached.preferred // the station that last reported completely carries over
		}
		p.mu.Lock()
		p.cache[k] = g
		p.mu.Unlock()
		return g, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*gridInfo), nil
}

// Retain drops every location the caller no longer tracks from the grid
// cache and the gridpoint memo (Q5, L4-F7): the structures follow the
// location set instead of growing with every lookup ever made.
func (p *Provider) Retain(refs []snapshot.LocationRef) {
	keep := make(map[snapshot.LocationKey]bool, len(refs))
	for _, r := range refs {
		keep[snapshot.Key(r)] = true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	live := map[string]bool{}
	for k, g := range p.cache {
		if !keep[k] {
			delete(p.cache, k)
			continue
		}
		live[g.gridURL] = true
	}
	for u := range p.grids {
		if !live[u] {
			delete(p.grids, u)
		}
	}
}

// resolvePoints performs the /points + /stations round trip for one location.
func (p *Provider) resolvePoints(ctx context.Context, ref snapshot.LocationRef) (*gridInfo, error) {
	var points struct {
		Properties struct {
			Forecast            string `json:"forecast"`
			ForecastHourly      string `json:"forecastHourly"`
			ForecastGridData    string `json:"forecastGridData"`
			ObservationStations string `json:"observationStations"`
			ForecastZone        string `json:"forecastZone"`
			County              string `json:"county"`
			FireWeatherZone     string `json:"fireWeatherZone"`
			TimeZone            string `json:"timeZone"`
		} `json:"properties"`
	}
	u := fmt.Sprintf("%s/points/%.4f,%.4f", p.base, ref.Lat, ref.Lon)
	if _, err := p.client.GetJSON(ctx, u, &points); err != nil {
		return nil, fmt.Errorf("resolving %s: %w", ref.Label, err)
	}
	g := &gridInfo{
		forecastURL: p.rebase(points.Properties.Forecast),
		hourlyURL:   p.rebase(points.Properties.ForecastHourly),
		gridURL:     p.rebase(points.Properties.ForecastGridData),
		stationsURL: p.rebase(points.Properties.ObservationStations),
		fireZone:    lastSegment(points.Properties.FireWeatherZone),
		timeZone:    points.Properties.TimeZone,
	}
	for _, zoneURL := range []string{points.Properties.ForecastZone, points.Properties.County} {
		if id := lastSegment(zoneURL); id != "" {
			g.zones = append(g.zones, id)
		}
	}
	if err := invariant.Check(len(g.zones) > 0, "nws: point resolved with no alert zones — M3 coverage would silently break for "+ref.Label); err != nil {
		return nil, err
	}

	var stations struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"` // [lon, lat]
			} `json:"geometry"`
			Properties struct {
				StationIdentifier string `json:"stationIdentifier"`
			} `json:"properties"`
		} `json:"features"`
	}
	if _, err := p.client.GetJSON(ctx, fmt.Sprintf("%s?limit=%d", g.stationsURL, stationCandidates), &stations); err != nil {
		return nil, fmt.Errorf("resolving stations for %s: %w", ref.Label, err)
	}
	if err := invariant.Check(len(stations.Features) > 0, "nws: no observation stations for "+ref.Label); err != nil {
		return nil, err
	}
	for _, f := range stations.Features {
		st := obsStation{id: f.Properties.StationIdentifier}
		if c := f.Geometry.Coordinates; len(c) == 2 {
			st.lon, st.lat = c[0], c[1]
		}
		if st.id != "" {
			g.stations = append(g.stations, st)
		}
	}
	return g, nil
}

// stationOrder is the obs fallback chain: the station that last reported
// completely first, then nearest-first.
func (p *Provider) stationOrder(g *gridInfo) []obsStation {
	p.mu.Lock()
	pref := g.preferred
	p.mu.Unlock()
	if pref == "" {
		return g.stations
	}
	out := make([]obsStation, 0, len(g.stations))
	for _, st := range g.stations {
		if st.id == pref {
			out = append([]obsStation{st}, out...)
		} else {
			out = append(out, st)
		}
	}
	return out
}

func (p *Provider) markPreferred(g *gridInfo, id string) {
	p.mu.Lock()
	g.preferred = id
	p.mu.Unlock()
}
