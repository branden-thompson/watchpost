// Package nws is the National Weather Service provider — the mandatory US
// source and the harmonization tie-break authority (T-E′, OQ-9).
//
// Endpoint flow per AI-1 (live-probed): /points/{lat},{lon} resolves the
// gridpoint, station list, and alert zones (cached for the process lifetime —
// grid data changes ~never; a scheduled daily refresh is a B3 long-running-mode
// task, tracked in the B1 ledger); observations come from the nearest station
// that reports a complete observation (4-station fallback chain — UAT 59);
// forecasts from the gridpoint endpoints; alerts from ONE batched
// /alerts/active?zone=... call covering every location's forecastZone AND
// county UGC (the M3 dual-UGC rule). All values normalize to SI (§10.1):
// wmoUnit conversions here, never at render.
package nws

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/tz"
)

// Attribution is the NWS credit line for the About view (OQ-15).
const Attribution = "NOAA National Weather Service (api.weather.gov)" // public domain

// stationCandidates is how many nearest observation stations a location
// keeps as an obs fallback chain (B3 UAT 59: Carlsbad's nearest station is
// a mesonet site with no sky condition and an intermittent temperature).
const stationCandidates = 4

// fetchConcurrency bounds per-location fan-out inside one Fetch (UAT 59).
// The httpx token bucket remains the single politeness governor; this only
// stops one slow station from serializing the whole batch.
const fetchConcurrency = 6

// Provider implements snapshot.Provider for api.weather.gov.
type Provider struct {
	client *httpx.Client
	base   string // e.g. https://api.weather.gov (test servers override)

	mu    sync.Mutex
	cache map[snapshot.LocationKey]*gridInfo
	sf    singleflight.Group // one points resolution per key across concurrent tiers
}

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
}

// New builds the provider. base "" means the production API.
func New(client *httpx.Client, base string) *Provider {
	if base == "" {
		base = "https://api.weather.gov"
	}
	return &Provider{client: client, base: base, cache: map[snapshot.LocationKey]*gridInfo{}}
}

// ID implements snapshot.Provider.
func (p *Provider) ID() string { return "nws" }

// Domains implements snapshot.Provider: NWS feeds weather AND alerts.
func (p *Provider) Domains() []string { return []string{"weather", "alerts"} }

// Fetch implements snapshot.Provider for one scheduled unit.
func (p *Provider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	frag := snapshot.Fragment{Provider: "nws", Kind: req.Kind, FetchedAt: time.Now().UTC(), PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	if err := invariant.Check(len(req.Locations) > 0, "nws: Fetch requires at least one location"); err != nil {
		return frag, err
	}
	switch req.Kind {
	case snapshot.KindObs:
		frag.PerLocation, frag.Err = snapshot.FetchEach(ctx, req.Locations, fetchConcurrency, p.fetchObs)
	case snapshot.KindForecast:
		frag.PerLocation, frag.Err = snapshot.FetchEach(ctx, req.Locations, fetchConcurrency, p.fetchForecast)
	case snapshot.KindForecastHourly:
		frag.PerLocation, frag.Err = snapshot.FetchEach(ctx, req.Locations, fetchConcurrency, p.fetchHourly)
	case snapshot.KindAlerts:
		if err := p.fetchAlerts(ctx, req.Locations, &frag); err != nil {
			frag.Err = err
		}
	default:
		return frag, fmt.Errorf("nws does not serve fetch kind %d", req.Kind)
	}
	return frag, nil
}

// FireWeatherZone resolves the NWS fire weather zone for a location (D-22:
// the app INFERS fire-zone membership from the user's own locations — a
// person cannot be expected to know their zone). "" when unresolvable.
func (p *Provider) FireWeatherZone(ctx context.Context, ref snapshot.LocationRef) string {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return "" // setup degrades to the generic prompt; never blocks
	}
	return g.fireZone
}

// ForecastZone is the location's forecast zone UGC ("CAZ043") from the
// cached point resolution — the synthesized broadcast reads only that
// zone's block of a product (UAT 81). "" when unresolvable.
func (p *Provider) ForecastZone(ctx context.Context, ref snapshot.LocationRef) string {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return ""
	}
	for _, z := range g.zones {
		if len(z) == 6 && z[2] == 'Z' {
			return z
		}
	}
	return ""
}

// CountyUGC is the location's county UGC ("CAC073") from the cached point
// resolution — the radio resolver turns it into a SAME code (B4). "" when
// unresolvable.
func (p *Provider) CountyUGC(ctx context.Context, ref snapshot.LocationRef) string {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return ""
	}
	for _, z := range g.zones {
		if len(z) == 6 && z[2] == 'C' {
			return z
		}
	}
	return ""
}

// Office is the forecast office (CWA id, e.g. "SGX") that issues the
// location's products — from the cached gridpoint URL (B4 synth). "" when
// unresolvable.
func (p *Provider) Office(ctx context.Context, ref snapshot.LocationRef) string {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(g.gridURL, p.base), "/") // /gridpoints/SGX/54,34
	if len(parts) >= 3 && parts[1] == "gridpoints" {
		return parts[2]
	}
	return ""
}

// --- resolve (points -> grid/stations/zones) ---

// CachedGrids reports how many locations' gridpoint resolutions are held
// (a structure the diagnostic dump watches; bounded on removal in Q5 — L4-F7).
func (p *Provider) CachedGrids() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.cache)
}

func (p *Provider) resolve(ctx context.Context, ref snapshot.LocationRef) (*gridInfo, error) {
	k := snapshot.Key(ref)
	p.mu.Lock()
	if g, ok := p.cache[k]; ok {
		p.mu.Unlock()
		return g, nil
	}
	p.mu.Unlock()
	// Singleflight (UAT 59): the alerts/obs/forecast tiers all resolve the
	// same location at launch — one points+stations round trip serves them.
	v, err, _ := p.sf.Do(string(k), func() (any, error) {
		g, err := p.resolvePoints(ctx, ref)
		if err != nil {
			return nil, err
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

// rebase rewrites api.weather.gov URLs onto p.base, and passes through URLs
// already relative to a BASE placeholder (test fixtures).
func (p *Provider) rebase(raw string) string {
	if strings.HasPrefix(raw, "BASE/") {
		return p.base + strings.TrimPrefix(raw, "BASE")
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return p.base + u.Path
	}
	return raw
}

func lastSegment(u string) string {
	i := strings.LastIndex(u, "/")
	if i < 0 || i == len(u)-1 {
		return ""
	}
	return u[i+1:]
}

// --- observations ---

type quantity struct {
	UnitCode       string   `json:"unitCode"`
	Value          *float64 `json:"value"`
	QualityControl string   `json:"qualityControl"`
}

// fetchObs walks the station fallback chain until one reports a COMPLETE
// observation (temperature + sky condition — UAT 59); failing that, the
// best partial observation (one with a temperature) is used.
func (p *Provider) fetchObs(ctx context.Context, ref snapshot.LocationRef) (snapshot.PartialData, error) {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return snapshot.PartialData{}, err
	}
	var best *snapshot.Conditions
	var lastErr error
	for _, st := range p.stationOrder(g) {
		c, err := p.stationObs(ctx, st, ref)
		if err != nil {
			lastErr = err
			continue
		}
		if c.Temp != nil && c.Condition != "unknown" {
			p.markPreferred(g, st.id)
			return snapshot.PartialData{Current: c}, nil
		}
		if best == nil || (best.Temp == nil && c.Temp != nil) {
			best = c
		}
	}
	if best != nil {
		return snapshot.PartialData{Current: best}, nil
	}
	return snapshot.PartialData{}, fmt.Errorf("observations for %s: %w", ref.Label, lastErr)
}

// stationObs reads one station's latest observation, normalized to SI, with
// the station's distance from the location recorded in Source (UAT 60).
func (p *Provider) stationObs(ctx context.Context, st obsStation, ref snapshot.LocationRef) (*snapshot.Conditions, error) {
	stationID := st.id
	var obs struct {
		Properties struct {
			Timestamp          time.Time `json:"timestamp"`
			TextDescription    string    `json:"textDescription"`
			Temperature        quantity  `json:"temperature"`
			Dewpoint           quantity  `json:"dewpoint"`
			WindSpeed          quantity  `json:"windSpeed"`
			WindGust           quantity  `json:"windGust"`
			WindDirection      quantity  `json:"windDirection"`
			BarometricPressure quantity  `json:"barometricPressure"`
			RelativeHumidity   quantity  `json:"relativeHumidity"`
			Visibility         quantity  `json:"visibility"`
			HeatIndex          quantity  `json:"heatIndex"`
		} `json:"properties"`
	}
	u := fmt.Sprintf("%s/stations/%s/observations/latest", p.base, stationID)
	if _, err := p.client.GetJSON(ctx, u, &obs); err != nil {
		return nil, fmt.Errorf("station %s: %w", stationID, err)
	}
	pr := obs.Properties
	return &snapshot.Conditions{
		ObservedAt:  pr.Timestamp,
		Temp:        toSI(pr.Temperature),
		Feels:       toSI(pr.HeatIndex),
		Dewpoint:    toSI(pr.Dewpoint),
		HumidityPct: toSI(pr.RelativeHumidity),
		Pressure:    toSI(pr.BarometricPressure),
		Wind:        toSI(pr.WindSpeed),
		WindGust:    toSI(pr.WindGust),
		WindDirDeg:  toSI(pr.WindDirection),
		Visibility:  toSI(pr.Visibility),
		Condition:   conditionCode(pr.TextDescription),
		Source: snapshot.SourceInfo{
			Provider:       "nws",
			ModelOrStation: stationID,
			DistanceKm:     stationDistance(st, ref),
			IssuedAt:       pr.Timestamp,
		},
	}, nil
}

// stationDistance is the station's great-circle distance from the location;
// nil when the station list carried no geometry (never a fake 0 — null parity).
func stationDistance(st obsStation, ref snapshot.LocationRef) *float64 {
	if st.lat == 0 && st.lon == 0 {
		return nil
	}
	d := geo.HaversineKM(ref.Lat, ref.Lon, st.lat, st.lon)
	return &d
}

// toSI converts a wmoUnit quantity to the snapshot's SI convention.
// Unknown units return nil rather than a wrong number (null-parity rule).
func toSI(q quantity) *float64 {
	if q.Value == nil {
		return nil
	}
	v := *q.Value
	switch q.UnitCode {
	case "wmoUnit:degC", "wmoUnit:percent", "wmoUnit:degree_(angle)", "wmoUnit:m":
		// already target unit
	case "wmoUnit:degF":
		v = (v - 32) * 5 / 9
	case "wmoUnit:km_h-1":
		v = v / 3.6
	case "wmoUnit:m_s-1":
		// already m/s
	case "wmoUnit:Pa":
		v = v / 100 // -> hPa
	case "wmoUnit:hPa":
		// already hPa
	default:
		return nil
	}
	return &v
}

// --- forecast ---

type period struct {
	StartTime                  time.Time `json:"startTime"`
	IsDaytime                  bool      `json:"isDaytime"`
	Temperature                float64   `json:"temperature"`
	TemperatureUnit            string    `json:"temperatureUnit"`
	ProbabilityOfPrecipitation quantity  `json:"probabilityOfPrecipitation"`
	WindSpeed                  string    `json:"windSpeed"`
	ShortForecast              string    `json:"shortForecast"`
}

// periodsDoc is the /forecast and /forecast/hourly payload shape.
type periodsDoc struct {
	Properties struct {
		Periods []period `json:"periods"`
	} `json:"properties"`
}

// fetchForecast is the daily forecast (KindForecast): 12-hour periods
// folded into calendar days, holes filled from the gridpoint.
func (p *Provider) fetchForecast(ctx context.Context, ref snapshot.LocationRef) (snapshot.PartialData, error) {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return snapshot.PartialData{}, err
	}
	var daily periodsDoc
	if _, err := p.client.GetJSON(ctx, g.forecastURL, &daily); err != nil {
		return snapshot.PartialData{}, fmt.Errorf("forecast for %s: %w", ref.Label, err)
	}
	pd := snapshot.PartialData{Daily: foldDaily(daily.Properties.Periods)}
	p.fillDailyFromGrid(ctx, g, pd.Daily)
	return pd, nil
}

// fetchHourly is the hourly forecast (KindForecastHourly, UAT 72 — its own
// tier: 162 KB per location, so the RECENT list hydrates it on demand).
func (p *Provider) fetchHourly(ctx context.Context, ref snapshot.LocationRef) (snapshot.PartialData, error) {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return snapshot.PartialData{}, err
	}
	var hourly periodsDoc
	if _, err := p.client.GetJSON(ctx, g.hourlyURL, &hourly); err != nil {
		return snapshot.PartialData{}, fmt.Errorf("hourly forecast for %s: %w", ref.Label, err)
	}
	pd := snapshot.PartialData{}
	for _, h := range hourly.Properties.Periods {
		t := tempC(h.Temperature, h.TemperatureUnit)
		pd.Hourly = append(pd.Hourly, snapshot.Hourly{
			Time:       h.StartTime,
			Temp:       &t,
			PrecipProb: h.ProbabilityOfPrecipitation.Value,
			Wind:       windFromText(h.WindSpeed),
			Condition:  conditionCode(h.ShortForecast),
		})
	}
	return pd, nil
}

// fillDailyFromGrid fills a day's missing HIGH/LOW from the raw gridpoint
// maxTemperature/minTemperature series (B3 UAT 71): /forecast drops a
// day's daytime period once local evening starts, so TODAY's HIGH read
// "n/a" east of wherever 6 PM had passed. The gridpoint keeps the value
// all day, and nws-marine already fetches it — the client cache makes this
// one download per grid per cycle. A nicety: any failure leaves the hole.
func (p *Provider) fillDailyFromGrid(ctx context.Context, g *gridInfo, daily []snapshot.Daily) {
	if g.gridURL == "" || !hasTempHole(daily) {
		return
	}
	var grid struct {
		Properties struct {
			Max gridSeries `json:"maxTemperature"`
			Min gridSeries `json:"minTemperature"`
		} `json:"properties"`
	}
	if _, err := p.client.GetJSON(ctx, g.gridURL, &grid); err != nil {
		return
	}
	tz, err := tz.Location(g.timeZone)
	if err != nil {
		tz = time.UTC
	}
	for i := range daily {
		d := &daily[i]
		if d.TempMax == nil {
			if v, ok := grid.Properties.Max.extremeOn(d.Date, tz, true); ok {
				d.TempMax, d.FillFrom = &v, fillNote(d.FillFrom, "temp_max")
			}
		}
		if d.TempMin == nil {
			if v, ok := grid.Properties.Min.extremeOn(d.Date, tz, false); ok {
				d.TempMin, d.FillFrom = &v, fillNote(d.FillFrom, "temp_min")
			}
		}
	}
}

func hasTempHole(daily []snapshot.Daily) bool {
	for _, d := range daily {
		if d.TempMax == nil || d.TempMin == nil {
			return true
		}
	}
	return false
}

func fillNote(m map[string]string, field string) map[string]string {
	if m == nil {
		m = map[string]string{}
	}
	m[field] = "nws:gridpoint"
	return m
}

// extremeOn is the max (or min) of a series' values whose validity starts
// on date in tz, in °C rounded to a tenth; false when the date has none.
func (s gridSeries) extremeOn(date string, tz *time.Location, wantMax bool) (float64, bool) {
	best, found := 0.0, false
	for _, v := range s.Values {
		if v.Value == nil {
			continue
		}
		start, err := time.Parse(time.RFC3339, strings.SplitN(v.ValidTime, "/", 2)[0])
		if err != nil || start.In(tz).Format("2006-01-02") != date {
			continue
		}
		val := *v.Value
		if s.UOM == "wmoUnit:degF" {
			val = (val - 32) * 5 / 9
		}
		if !found || (wantMax && val > best) || (!wantMax && val < best) {
			best, found = val, true
		}
	}
	return roundTenth(best), found
}

// foldDaily folds NWS 12-hour day/night periods into calendar days.
func foldDaily(periods []period) []snapshot.Daily {
	byDate := map[string]*snapshot.Daily{}
	var order []string
	for _, per := range periods {
		date := per.StartTime.Format("2006-01-02")
		d, ok := byDate[date]
		if !ok {
			d = &snapshot.Daily{Date: date}
			byDate[date] = d
			order = append(order, date)
		}
		t := tempC(per.Temperature, per.TemperatureUnit)
		if per.IsDaytime {
			d.TempMax = &t
			d.Condition = conditionCode(per.ShortForecast)
		} else {
			d.TempMin = &t
			if d.Condition == "" {
				d.Condition = conditionCode(per.ShortForecast)
			}
		}
		if v := per.ProbabilityOfPrecipitation.Value; v != nil {
			if d.PrecipProb == nil || *v > *d.PrecipProb {
				d.PrecipProb = v
			}
		}
	}
	sort.Strings(order)
	out := make([]snapshot.Daily, 0, len(order))
	for _, date := range order {
		out = append(out, *byDate[date])
	}
	return out
}

func tempC(v float64, unit string) float64 {
	if unit == "F" {
		return roundTenth((v - 32) * 5 / 9)
	}
	return roundTenth(v)
}

func roundTenth(v float64) float64 {
	if v < 0 {
		return float64(int(v*10-0.5)) / 10
	}
	return float64(int(v*10+0.5)) / 10
}

// windFromText parses NWS textual wind ("5 to 10 mph") to m/s (upper bound).
func windFromText(s string) *float64 {
	fields := strings.Fields(s)
	var mph float64
	found := false
	for i, f := range fields {
		if f == "mph" && i > 0 {
			if _, err := fmt.Sscanf(fields[i-1], "%f", &mph); err == nil {
				found = true
			}
		}
	}
	if !found {
		return nil
	}
	v := roundTenth(mph * 0.44704)
	return &v
}

// conditionCode maps NWS text to the closed condition enum (§10.1).
func conditionCode(text string) string {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "thunder"):
		return "thunderstorm"
	case strings.Contains(t, "snow"), strings.Contains(t, "flurr"):
		return "snow"
	case strings.Contains(t, "rain"), strings.Contains(t, "shower"), strings.Contains(t, "drizzle"):
		return "rain"
	case strings.Contains(t, "fog"), strings.Contains(t, "mist"), strings.Contains(t, "haze"), strings.Contains(t, "smoke"):
		return "fog"
	case strings.Contains(t, "partly"), strings.Contains(t, "mostly sunny"), strings.Contains(t, "mostly clear"):
		return "partly_cloudy"
	case strings.Contains(t, "cloud"), strings.Contains(t, "overcast"):
		return "cloudy"
	case strings.Contains(t, "sunny"), strings.Contains(t, "clear"), strings.Contains(t, "fair"):
		return "clear"
	default:
		return "unknown"
	}
}

// --- alerts ---

// alertProps is the CAP properties payload from /alerts/active.
type alertProps struct {
	ID          string     `json:"id"`
	Event       string     `json:"event"`
	Severity    string     `json:"severity"`
	Urgency     string     `json:"urgency"`
	Certainty   string     `json:"certainty"`
	MessageType string     `json:"messageType"`
	Sent        time.Time  `json:"sent"`
	Effective   time.Time  `json:"effective"`
	Onset       *time.Time `json:"onset"`
	Expires     time.Time  `json:"expires"`
	Ends        *time.Time `json:"ends"`
	References  []struct {
		ID string `json:"@id"`
	} `json:"references"`
	AffectedZones []string `json:"affectedZones"`
	AreaDesc      string   `json:"areaDesc"`
	Headline      string   `json:"headline"`
	Description   string   `json:"description"`
	Instruction   string   `json:"instruction"`
}

func (p *Provider) fetchAlerts(ctx context.Context, refs []snapshot.LocationRef, frag *snapshot.Fragment) error {
	// Collect every location's zones (dual-UGC: forecastZone + county — M3).
	zoneToKeys := map[string][]snapshot.LocationKey{}
	var zones []string
	for _, ref := range refs {
		g, err := p.resolve(ctx, ref)
		if err != nil {
			return err
		}
		k := snapshot.Key(ref)
		for _, z := range g.zones {
			if _, seen := zoneToKeys[z]; !seen {
				zones = append(zones, z)
			}
			zoneToKeys[z] = append(zoneToKeys[z], k)
		}
	}
	sort.Strings(zones)
	var payload struct {
		Features []struct {
			Properties alertProps `json:"properties"`
		} `json:"features"`
	}
	u := fmt.Sprintf("%s/alerts/active?status=actual&zone=%s", p.base, strings.Join(zones, ","))
	if _, err := p.client.GetJSON(ctx, u, &payload); err != nil {
		return fmt.Errorf("alerts: %w", err)
	}
	perKey := map[snapshot.LocationKey][]snapshot.Alert{}
	for _, f := range payload.Features {
		mapAlert(f.Properties, zoneToKeys, perKey)
	}
	for _, ref := range refs {
		k := snapshot.Key(ref)
		alerts := perKey[k]
		if alerts == nil {
			alerts = []snapshot.Alert{} // non-nil: "fetched, none active" replaces stale sets
		}
		frag.PerLocation[k] = snapshot.PartialData{Alerts: alerts}
	}
	return nil
}

// mapAlert converts one CAP feature and attaches it to every watched location
// whose zones it affects (deduped per location).
func mapAlert(pr alertProps, zoneToKeys map[string][]snapshot.LocationKey, perKey map[snapshot.LocationKey][]snapshot.Alert) {
	if err := invariant.Check(pr.ID != "", "CAP alert without an id cannot be deduplicated"); err != nil {
		pr.ID = "no-id:" + pr.Headline // never drop an alert silently (RS-10)
	}
	a := snapshot.Alert{
		ID:          pr.ID,
		Event:       pr.Event,
		Severity:    strings.ToLower(pr.Severity),
		Urgency:     strings.ToLower(pr.Urgency),
		Certainty:   strings.ToLower(pr.Certainty),
		MessageType: strings.ToLower(pr.MessageType),
		Sent:        pr.Sent, Effective: pr.Effective, Onset: pr.Onset,
		Expires: pr.Expires, Ends: pr.Ends,
		AreaDesc: pr.AreaDesc, Headline: pr.Headline,
		Description: pr.Description, Instruction: pr.Instruction,
		Source: snapshot.SourceInfo{Provider: "nws", IssuedAt: pr.Sent},
	}
	for _, r := range pr.References {
		a.References = append(a.References, r.ID)
	}
	// Two passes (B1 red-team #1): the full zone list must be complete BEFORE
	// any location receives its copy, or early-matched locations get a
	// truncated CAP AffectedZones.
	for _, zURL := range pr.AffectedZones {
		a.AffectedZones = append(a.AffectedZones, lastSegment(zURL))
	}
	matched := map[snapshot.LocationKey]bool{}
	for _, z := range a.AffectedZones {
		for _, k := range zoneToKeys[z] {
			if !matched[k] {
				matched[k] = true
				perKey[k] = append(perKey[k], a)
			}
		}
	}
}
