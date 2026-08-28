package globalfeed

import (
	"context"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// nhcTTL: NHC issues advisories every few hours.
const nhcTTL = 30 * time.Minute

// NHC reads the National Hurricane Center current-storms feed (Atlantic and
// Eastern Pacific basins; keyless). W-Pacific typhoons are not carried (D1).
type NHC struct {
	client *httpx.Client
	base   string
}

// NewNHC builds the source; base "" is the production CurrentStorms feed.
func NewNHC(client *httpx.Client, base string) *NHC {
	if base == "" {
		base = "https://www.nhc.noaa.gov/CurrentStorms.json"
	}
	return &NHC{client: client, base: base}
}

func (n *NHC) Name() string { return "NHC" }

type nhcFeed struct {
	ActiveStorms []struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		Classification string  `json:"classification"`
		LatitudeNum    float64 `json:"latitudeNumeric"`
		LongitudeNum   float64 `json:"longitudeNumeric"`
		LastUpdate     string  `json:"lastUpdate"`
	} `json:"activeStorms"`
}

func (n *NHC) Fetch(ctx context.Context) ([]Event, error) {
	var feed nhcFeed
	if _, err := n.client.GetJSON(ctx, n.base, &feed, httpx.TTL(nhcTTL)); err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(feed.ActiveStorms))
	for _, s := range feed.ActiveStorms {
		typ, sev, ok := tropicalClass(s.Classification)
		if !ok || s.ID == "" {
			continue // not an active cyclone class (a low/disturbance/post-tropical) — skip
		}
		at, _ := time.Parse(time.RFC3339, s.LastUpdate)
		if at.IsZero() {
			at = time.Now() // a malformed advisory time is "as of now", not year 1 (P4 F6)
		}
		out = append(out, Event{
			ID:       s.ID,
			Class:    ClassTropical,
			Severity: sev,
			Type:     typ,
			Place:    tropicalBasin(s.ID),
			Lat:      s.LatitudeNum,
			Lon:      s.LongitudeNum,
			HasPoint: true,
			At:       at.UTC(),
			Source:   n.Name(),
		})
	}
	return out, nil
}

// tropicalClass maps an NHC classification to the spoken type and severity;
// ok is false for classes the ticker does not carry (lows, disturbances,
// post-tropical). HU red, TS orange, TD/PTC yellow.
func tropicalClass(c string) (typ string, sev Severity, ok bool) {
	switch strings.ToUpper(strings.TrimSpace(c)) {
	case "HU":
		return "Hurricane", SevRed, true
	case "TS":
		return "Tropical Storm", SevOrange, true
	case "TD":
		return "Tropical Depression", SevYellow, true
	case "PTC":
		return "Potential Tropical Cyclone", SevYellow, true
	}
	return "", SevYellow, false
}

// tropicalBasin names the storm's basin from its id prefix (al = Atlantic,
// ep = Eastern Pacific, cp = Central Pacific).
func tropicalBasin(id string) string {
	switch strings.ToLower(id[:min(2, len(id))]) {
	case "al":
		return "the Atlantic"
	case "ep":
		return "the Eastern Pacific"
	case "cp":
		return "the Central Pacific"
	}
	return "the open ocean"
}
