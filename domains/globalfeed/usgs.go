package globalfeed

import (
	"context"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Source is one global feed the ticker reads. Fetch returns the feed's current
// events (unlocated — Locate ties each to a representative place); an error
// leaves the ticker's prior stack in place (the pipeline retries).
type Source interface {
	Name() string
	Fetch(ctx context.Context) ([]Event, error)
}

// Attribution lines (all public domain / keyless).
const (
	AttrUSGS = "USGS Earthquake Hazards Program, earthquake.usgs.gov"
	AttrNHC  = "NOAA National Hurricane Center, nhc.noaa.gov"
	AttrNWS  = "NOAA National Weather Service, weather.gov"
)

// usgsTTL: the significant feed is a summary file refreshed every few minutes.
const usgsTTL = 5 * time.Minute

// USGS reads the USGS significant-earthquakes summary feed (global, keyless).
type USGS struct {
	client *httpx.Client
	base   string
}

// NewUSGS builds the source; base "" is the production significant_week feed.
func NewUSGS(client *httpx.Client, base string) *USGS {
	if base == "" {
		base = "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/significant_week.geojson"
	}
	return &USGS{client: client, base: base}
}

func (u *USGS) Name() string { return "USGS" }

// usgsFeed is the slice of the summary GeoJSON the ticker reads.
type usgsFeed struct {
	Features []struct {
		ID         string `json:"id"`
		Properties struct {
			Mag     *float64 `json:"mag"`
			Place   string   `json:"place"`
			Time    int64    `json:"time"` // epoch ms
			Type    string   `json:"type"`
			Tsunami int      `json:"tsunami"`
		} `json:"properties"`
		Geometry struct {
			Coordinates []float64 `json:"coordinates"` // [lon, lat, depth]
		} `json:"geometry"`
	} `json:"features"`
}

func (u *USGS) Fetch(ctx context.Context) ([]Event, error) {
	var feed usgsFeed
	if _, err := u.client.GetJSON(ctx, u.base, &feed, httpx.TTL(usgsTTL)); err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(feed.Features))
	for _, f := range feed.Features {
		g := f.Geometry.Coordinates
		if f.ID == "" || f.Properties.Mag == nil || len(g) < 2 {
			continue
		}
		mag := *f.Properties.Mag
		at := time.UnixMilli(f.Properties.Time).UTC()
		if f.Properties.Time == 0 {
			at = time.Now() // a missing epoch is "as of now", not 1970 (P4 F6)
		}
		out = append(out, Event{
			ID:       f.ID,
			Class:    ClassQuake,
			Severity: quakeSeverity(mag, f.Properties.Tsunami != 0),
			Type:     clampField(quakeType(f.Properties.Type)), // bounded feed field (P4 F5)
			Place:    clampField(f.Properties.Place),
			Lat:      g[1],
			Lon:      g[0],
			HasPoint: true,
			At:       at,
			Source:   u.Name(),
		})
	}
	return out, nil
}

// quakeSeverity: a great quake or any tsunami reads red; a strong quake orange;
// the rest yellow (§plan severity map).
func quakeSeverity(mag float64, tsunami bool) Severity {
	switch {
	case tsunami || mag >= 6.5:
		return SevRed
	case mag >= 5.5:
		return SevOrange
	default:
		return SevYellow
	}
}

// quakeType is the spoken event type; USGS "type" is usually "earthquake" but
// the significant feed also carries "landslide"/"explosion" etc.
func quakeType(t string) string {
	t = strings.TrimSpace(t)
	if t == "" || strings.EqualFold(t, "earthquake") {
		return "Earthquake"
	}
	return strings.ToUpper(t[:1]) + t[1:]
}
