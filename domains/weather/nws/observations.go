package nws

// observations.go — observations: the station fallback chain, SI conversion, wind and condition text parsing. Split from provider.go by the quality pass (Q2, pure move).

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

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
