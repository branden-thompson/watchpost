package snapshot

// harmonize.go — the pure merge and harmonize steps that shape a published Location (finalize, normalize, harmonize, marine fill, sun times). Split from assembler.go by the quality pass (Q2, pure move).

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/astro"
	"github.com/branden-thompson/watchpost/platform/tz"
)

// finalize derives a location's published view from its provider sections:
// harmonize across providers, rehydrate sparse observations from the
// forecast, compute sun times, then normalize collections for machine mode.
func finalize(loc *Location, providers []string, now time.Time) {
	harmonize(loc, providers)
	rehydrateFromForecast(loc, now)
	fillSunTimes(loc)
	normalizeCollections(loc)
}

// normalizeCollections guarantees the machine-mode contract: collections are
// always arrays (possibly empty), never null — agents get stable types
// (AI-10 §1 conventions; the null-parity rule covers VALUE fields only).
func normalizeCollections(loc *Location) {
	if loc.Alerts == nil {
		loc.Alerts = []Alert{}
	}
	if loc.Hourly == nil {
		loc.Hourly = []Hourly{}
	}
	if loc.Daily == nil {
		loc.Daily = []Daily{}
	}
	if loc.Fire.Hotspots == nil {
		loc.Fire.Hotspots = []Hotspot{}
	}
	if loc.Fire.Incidents == nil {
		loc.Fire.Incidents = []Incident{}
	}
	if loc.ByProvider == nil {
		loc.ByProvider = map[string]Section{}
	}
	if loc.Radio.Source == "" { // the tuner block's closed sets never publish "" (REVIEW M3)
		loc.Radio.Source = "none"
	}
	if loc.Radio.Status == "" {
		loc.Radio.Status = "none"
	}
	normalizeMarine(loc.Marine)
	for _, sec := range loc.ByProvider { // every provider copy too: by_provider.*.marine.tides/currents are arrays, never null (REVIEW M1)
		normalizeMarine(sec.Marine)
	}
}

// normalizeMarine empties a marine block's nil collections (schema: arrays).
func normalizeMarine(m *Marine) {
	if m == nil {
		return
	}
	if m.Tides == nil {
		m.Tides = []TideEvent{}
	}
	if m.Currents == nil {
		m.Currents = []CurrentEvent{}
	}
}

// harmonize fills Location.Harmonized/Hourly/Daily per the OQ-9 rule: NWS wins
// outright; secondaries fill only nil fields (fill_from recorded); no blending.
// Provider order = configured order after "nws".
func harmonize(loc *Location, providerOrder []string) {
	order := make([]string, 0, len(providerOrder))
	for _, id := range providerOrder {
		if id == "nws" {
			order = append([]string{"nws"}, order...)
		} else {
			order = append(order, id)
		}
	}
	for _, id := range order {
		sec, ok := loc.ByProvider[id]
		if !ok || sec.Current == nil {
			continue
		}
		if loc.Harmonized.Source.Provider == "" {
			loc.Harmonized = *sec.Current
			continue
		}
		fillFrom(&loc.Harmonized, sec.Current, id)
	}
	// Hourly/Daily: first provider in order that has them (no cross-provider
	// splicing of time series).
	for _, id := range order {
		sec, ok := loc.ByProvider[id]
		if !ok {
			continue
		}
		if loc.Hourly == nil && len(sec.Hourly) > 0 {
			loc.Hourly = append([]Hourly(nil), sec.Hourly...)
		}
		if loc.Daily == nil && len(sec.Daily) > 0 {
			loc.Daily = append([]Daily(nil), sec.Daily...)
		}
	}
	harmonizeMarine(loc, order)
}

// keepOr returns have when set, else a copy of the fallback (single-deref
// value helper, P10-09 pointer discipline).
func keepOr(have, fallback *float64) *float64 {
	if have != nil || fallback == nil {
		return have
	}
	v := *fallback
	return &v
}

// fillForecast is the FillFrom provenance for values rehydrated from the
// location's own hourly forecast rather than a second provider.
const fillForecast = "forecast"

// rehydrateFromForecast fills a SPARSE observation's holes from the hourly
// forecast period covering now (B3 UAT 59): mesonet stations publish no sky
// condition and an intermittent temperature, which read as "UNKNOWN n/a".
// Observed values are never replaced; provenance is recorded; a location
// with no observation at all stays a loading state (the obs retry owns it).
func rehydrateFromForecast(loc *Location, now time.Time) {
	c := &loc.Harmonized
	if c.Source.Provider == "" || (c.Temp != nil && c.Condition != "" && c.Condition != "unknown") {
		return
	}
	h, ok := hourCovering(loc.Hourly, now)
	if !ok {
		return
	}
	record := func(field string) {
		if c.Source.FillFrom == nil {
			c.Source.FillFrom = map[string]string{}
		}
		c.Source.FillFrom[field] = fillForecast
	}
	if c.Temp == nil && h.Temp != nil {
		c.Temp = keepOr(nil, h.Temp)
		record("temp")
	}
	if (c.Condition == "" || c.Condition == "unknown") && h.Condition != "" && h.Condition != "unknown" {
		c.Condition = h.Condition
		record("condition_code")
	}
}

// hourCovering returns the forecast period whose hour contains now.
func hourCovering(hours []Hourly, now time.Time) (Hourly, bool) {
	for _, h := range hours {
		if !now.Before(h.Time) && now.Before(h.Time.Add(time.Hour)) {
			return h, true
		}
	}
	return Hourly{}, false
}

// fillSunTimes computes sunrise/sunset for every Daily row that lacks them
// (no provider carries them — B3 UAT 32.4): geometry from lat/lon in the
// location's timezone.
func fillSunTimes(loc *Location) {
	tz, err := tz.Location(loc.TZ)
	if err != nil || loc.TZ == "" {
		tz = time.UTC
	}
	for i := range loc.Daily {
		d := &loc.Daily[i]
		if !d.Sunrise.IsZero() && !d.Sunset.IsZero() {
			continue
		}
		date, perr := time.ParseInLocation("2006-01-02", d.Date, tz)
		if perr != nil {
			continue
		}
		if rise, set, ok := astro.SunTimes(loc.Lat, loc.Lon, date, tz); ok {
			d.Sunrise, d.Sunset = rise, set
		}
	}
}

// harmonizeMarine merges the coastal-waters section field-wise across
// providers in order (forecast provider first, buoy fills the rest — water
// temperature only ever comes from the buoy). nil stays nil inland.
func harmonizeMarine(loc *Location, order []string) {
	for _, id := range order {
		sec, ok := loc.ByProvider[id]
		if !ok || sec.Marine == nil {
			continue
		}
		if loc.Marine == nil {
			loc.Marine = sec.Marine.Clone()
			continue
		}
		fillMarine(loc.Marine, sec.Marine)
	}
}

// fillMarine copies src's non-nil fields into dst's nil fields (never
// replaces); buoy identity travels with the water temperature.
func fillMarine(dst, src *Marine) {
	dst.SwellHeight = keepOr(dst.SwellHeight, src.SwellHeight)
	dst.SwellDirDeg = keepOr(dst.SwellDirDeg, src.SwellDirDeg)
	dst.WaveHeight = keepOr(dst.WaveHeight, src.WaveHeight)
	dst.WavePeriod = keepOr(dst.WavePeriod, src.WavePeriod)
	dst.WindWaveHeight = keepOr(dst.WindWaveHeight, src.WindWaveHeight)
	dst.SecondarySwellHeight = keepOr(dst.SecondarySwellHeight, src.SecondarySwellHeight)
	dst.SecondarySwellDirDeg = keepOr(dst.SecondarySwellDirDeg, src.SecondarySwellDirDeg)
	dst.SecondaryPeriod = keepOr(dst.SecondaryPeriod, src.SecondaryPeriod)
	dst.WindSpeed = keepOr(dst.WindSpeed, src.WindSpeed)
	dst.WindGust = keepOr(dst.WindGust, src.WindGust)
	dst.WaterTemp = keepOr(dst.WaterTemp, src.WaterTemp)
	dst.BuoyDistanceKM = keepOr(dst.BuoyDistanceKM, src.BuoyDistanceKM)
	if dst.Buoy == "" {
		dst.Buoy = src.Buoy
	}
	if dst.ObservedAt.IsZero() {
		dst.ObservedAt = src.ObservedAt
	}
	fillTides(dst, src)
}

// fillTides copies the tide/current block (UAT 61) into a section that has
// none — the whole block travels together with its station identity.
func fillTides(dst, src *Marine) {
	dst.TideLevel = keepOr(dst.TideLevel, src.TideLevel)
	if len(dst.Tides) == 0 && len(src.Tides) > 0 {
		dst.Tides = append([]TideEvent(nil), src.Tides...)
		dst.TideStation, dst.TideStationKM = src.TideStation, keepOr(nil, src.TideStationKM)
	}
	if len(dst.Currents) == 0 && len(src.Currents) > 0 {
		dst.Currents = append([]CurrentEvent(nil), src.Currents...)
		dst.CurrentStation = src.CurrentStation
	}
}

// fillFrom copies src's non-nil fields into nil fields of dst, recording
// provenance in dst.Source.FillFrom (never replaces existing values).
// DEFERRED (§10.11, lands with multi-provider B5): the staleness cutoff —
// fill only when src.ObservedAt is within 2x the field's tier cadence. With a
// single provider (B1) there is nothing to fill from, so the gap is inert;
// the B5 harmonization goldens must cover it (tracked in the B1 ledger).
func fillFrom(dst, src *Conditions, srcID string) {
	record := func(field string) {
		if dst.Source.FillFrom == nil {
			dst.Source.FillFrom = map[string]string{}
		}
		dst.Source.FillFrom[field] = srcID
	}
	fill := func(d, s *float64, name string) *float64 {
		if d != nil || s == nil {
			return d
		}
		v := *s
		record(name)
		return &v
	}
	dst.Temp = fill(dst.Temp, src.Temp, "temp")
	dst.Feels = fill(dst.Feels, src.Feels, "feels")
	dst.Dewpoint = fill(dst.Dewpoint, src.Dewpoint, "dewpoint")
	dst.HumidityPct = fill(dst.HumidityPct, src.HumidityPct, "humidity_pct")
	dst.Pressure = fill(dst.Pressure, src.Pressure, "pressure")
	dst.Wind = fill(dst.Wind, src.Wind, "wind")
	dst.WindDirDeg = fill(dst.WindDirDeg, src.WindDirDeg, "wind_dir_deg")
	dst.WindGust = fill(dst.WindGust, src.WindGust, "wind_gust")
	dst.Precip1h = fill(dst.Precip1h, src.Precip1h, "precip_1h")
	dst.PrecipProb = fill(dst.PrecipProb, src.PrecipProb, "precip_prob_pct")
	dst.CloudPct = fill(dst.CloudPct, src.CloudPct, "cloud_pct")
	dst.Visibility = fill(dst.Visibility, src.Visibility, "visibility")
	dst.UVIndex = fill(dst.UVIndex, src.UVIndex, "uv_index")
	if dst.IsDay == nil && src.IsDay != nil {
		v := *src.IsDay
		dst.IsDay = &v
		record("is_day")
	}
	if dst.Condition == "" && src.Condition != "" {
		dst.Condition = src.Condition
		record("condition_code")
	}
}
