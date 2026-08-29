package globalfeed

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	memo   sourceMemo
}

// NewNHC builds the source; base "" is the production CurrentStorms feed.
func NewNHC(client *httpx.Client, base string) *NHC {
	if base == "" {
		base = "https://www.nhc.noaa.gov/CurrentStorms.json"
	}
	return &NHC{client: client, base: base}
}

func (n *NHC) Name() string { return "NHC" }

type nhcAdvisory struct {
	AdvNum   string `json:"advNum"`
	Issuance string `json:"issuance"`
	URL      string `json:"url"`
}

// nhcStorm is one entry of the feed.
type nhcStorm struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Classification string      `json:"classification"`
	BinNumber      string      `json:"binNumber"`
	Intensity      string      `json:"intensity"` // knots, as a string in the feed
	Pressure       string      `json:"pressure"`  // mb, as a string in the feed
	Latitude       string      `json:"latitude"`
	Longitude      string      `json:"longitude"`
	LatitudeNum    float64     `json:"latitudeNumeric"`
	LongitudeNum   float64     `json:"longitudeNumeric"`
	MovementDir    int         `json:"movementDir"`
	MovementSpeed  int         `json:"movementSpeed"`
	LastUpdate     string      `json:"lastUpdate"`
	PublicAdvisory nhcAdvisory `json:"publicAdvisory"`
	ForecastAdv    nhcAdvisory `json:"forecastAdvisory"`
	Discussion     nhcAdvisory `json:"forecastDiscussion"`
}

type nhcFeed struct {
	ActiveStorms []json.RawMessage `json:"activeStorms"` // decoded one by one: a malformed value skips ITS entry, never the source (REVIEW R5-C-12)
}

func (n *NHC) Fetch(ctx context.Context) ([]Event, error) {
	return n.memo.events(func() ([]byte, bool, error) {
		var body []byte
		hdr, err := n.client.GetJSON(ctx, n.base, &body, httpx.TTL(nhcTTL))
		return body, hdr == nil, err
	}, n.parse)
}

func (n *NHC) parse(body []byte) ([]Event, error) {
	var feed nhcFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		if n.client != nil {
			n.client.Forget(n.base)
		}
		return nil, fmt.Errorf("NHC: bad response body: %w", err)
	}
	out := make([]Event, 0, len(feed.ActiveStorms))
	for _, raw := range feed.ActiveStorms {
		var s nhcStorm
		if json.Unmarshal(raw, &s) != nil {
			continue // one bad entry (R5-C-12)
		}
		typ, sev, ok := tropicalClass(s.Classification)
		if !ok || s.ID == "" {
			continue // not an active cyclone class (a low/disturbance/post-tropical) — skip
		}
		at, _ := time.Parse(time.RFC3339, s.LastUpdate)
		if at.IsZero() {
			at = time.Now() // a malformed advisory time is "as of now", not year 1 (P4 F6)
		}
		advAt, _ := time.Parse(time.RFC3339, s.PublicAdvisory.Issuance)
		name := clampField(strings.TrimSpace(s.Name))
		out = append(out, Event{
			ID:       clampField(s.ID),
			Class:    ClassTropical,
			Severity: sev,
			Type:     typ,
			Name:     name, // SAM-D-14/20: the storm is announced by name
			Place:    tropicalBasin(s.ID),
			Lat:      s.LatitudeNum,
			Lon:      s.LongitudeNum,
			HasPoint: true,
			At:       at.UTC(),
			Source:   n.Name(),
			Tropical: &TropicalDetail{
				Name: name, Basin: tropicalBasin(s.ID), BinNumber: clampField(s.BinNumber),
				WindKt: clampNonNeg(atoiLoose(s.Intensity), 250), PressureMb: clampNonNeg(atoiLoose(s.Pressure), 1100),
				MoveDirDeg: clampNonNeg(s.MovementDir, 360), MoveSpeedKt: clampNonNeg(s.MovementSpeed, 100),
				LatText: clampField(s.Latitude), LonText: clampField(s.Longitude),
				AdvisoryNum: clampField(s.PublicAdvisory.AdvNum), AdvisoryAt: advAt.UTC(),
				ForecastNum: clampField(s.ForecastAdv.AdvNum), DiscussionNum: clampField(s.Discussion.AdvNum),
				AdvisoryURL: clampField(s.PublicAdvisory.URL),
			},
		})
	}
	return out, nil
}

// atoiLoose reads the feed's numeric strings ("45", "999"); anything else is 0.
func atoiLoose(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
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
