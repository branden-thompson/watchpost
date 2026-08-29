package globalfeed

import (
	"context"
	"encoding/json"
	"fmt"
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
	memo   sourceMemo
}

// NewUSGS builds the source; base "" is the production significant_week feed.
func NewUSGS(client *httpx.Client, base string) *USGS {
	if base == "" {
		base = "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/significant_week.geojson"
	}
	return &USGS{client: client, base: base}
}

func (u *USGS) Name() string { return "USGS" }

// usgsFeed is the slice of the summary GeoJSON the ticker and the severe
// window read (the render list of data-shape.md §4; the network/telemetry tail
// is not decoded in v1 — SAM-D-24 E-2).
// usgsFeature is one entry of the feed.
type usgsFeature struct {
	ID         string `json:"id"`
	Properties struct {
		Mag     *float64 `json:"mag"`
		MagType string   `json:"magType"`
		Place   string   `json:"place"`
		Time    int64    `json:"time"`    // epoch ms
		Updated int64    `json:"updated"` // epoch ms
		Type    string   `json:"type"`
		Title   string   `json:"title"`
		Alert   string   `json:"alert"`
		Status  string   `json:"status"`
		Tsunami int      `json:"tsunami"`
		Sig     int      `json:"sig"`
		Felt    *int     `json:"felt"`
		CDI     *float64 `json:"cdi"`
		MMI     *float64 `json:"mmi"`
		URL     string   `json:"url"`
		Detail  string   `json:"detail"`
	} `json:"properties"`
	Geometry struct {
		Coordinates []float64 `json:"coordinates"` // [lon, lat, depth]
	} `json:"geometry"`
}

type usgsFeed struct {
	Features []json.RawMessage `json:"features"` // decoded one by one: a malformed value skips ITS entry, never the source (REVIEW R5-C-12)
}

// Fetch reads the feed through the parse memo: the body comes from httpx
// (cache/TTL/conditional GET as before); it is decoded only when changed.
func (u *USGS) Fetch(ctx context.Context) ([]Event, error) {
	return u.memo.events(func() ([]byte, bool, error) {
		var body []byte
		hdr, err := u.client.GetJSON(ctx, u.base, &body, httpx.TTL(usgsTTL)) // read-only slice (the GetText contract)
		return body, hdr == nil, err
	}, u.parse)
}

// parse decodes one body into events (pure; the memo calls it on change).
func (u *USGS) parse(body []byte) ([]Event, error) {
	var feed usgsFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		if u.client != nil {
			u.client.Forget(u.base) // the GetJSON poison guard, kept on the raw-body path
		}
		return nil, fmt.Errorf("USGS: bad response body: %w", err)
	}
	out := make([]Event, 0, len(feed.Features))
	for _, raw := range feed.Features {
		var f usgsFeature
		if json.Unmarshal(raw, &f) != nil {
			continue // one bad entry (R5-C-12)
		}
		g := f.Geometry.Coordinates
		if f.ID == "" || f.Properties.Mag == nil || len(g) < 2 {
			continue
		}
		p := f.Properties
		mag := clampFloat(*p.Mag, -1, 12)
		at := time.UnixMilli(p.Time).UTC()
		if p.Time == 0 {
			at = time.Now() // a missing epoch is "as of now", not 1970 (P4 F6)
		}
		d := &QuakeDetail{
			Mag: &mag, MagType: clampField(p.MagType), Title: clampField(p.Title),
			Alert: pagerAlert(p.Alert), Sig: clampNonNeg(p.Sig, 5000), Status: clampField(p.Status),
			Tsunami: p.Tsunami != 0, URL: clampField(p.URL), DetailURL: clampField(p.Detail),
		}
		if len(g) >= 3 {
			d.DepthKm = clampFloat(g[2], 0, 1000)
		}
		if p.Felt != nil {
			d.Felt = clampNonNeg(*p.Felt, 1_000_000)
		}
		if p.CDI != nil {
			v := clampFloat(*p.CDI, 0, 12)
			d.CDI = &v
		}
		if p.MMI != nil {
			v := clampFloat(*p.MMI, 0, 12)
			d.MMI = &v
		}
		if p.Updated > 0 {
			d.UpdatedAt = time.UnixMilli(p.Updated).UTC()
		}
		out = append(out, Event{
			ID:       clampField(f.ID), // bounded like every other field: the key, the seen-store entry and the tape id (R3-D-02)
			Class:    ClassQuake,
			Severity: quakeSeverity(mag, p.Tsunami != 0),
			Type:     clampField(quakeType(p.Type)), // bounded feed field (P4 F5)
			Place:    clampField(p.Place),
			Lat:      g[1],
			Lon:      g[0],
			HasPoint: true,
			At:       at,
			Source:   u.Name(),
			Quake:    d,
		})
	}
	return out, nil
}

// pagerAlert validates the PAGER tier against its enum; anything else is "".
func pagerAlert(s string) string {
	switch v := strings.ToLower(strings.TrimSpace(s)); v {
	case "green", "yellow", "orange", "red":
		return v
	}
	return ""
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
