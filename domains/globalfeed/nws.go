package globalfeed

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// nwsTTL: warnings change quickly during an outbreak.
const nwsTTL = 2 * time.Minute

type severeEvent struct {
	event string
	sev   Severity
}

// severeEvents is the curated national query (US only): the warnings/watches
// the ticker carries, with their severity tier. The NWS `severity=` multi-param
// 400s (probed 2026-08-27), so the query filters by these event names instead.
// A function, not a global, per the codebase's table convention (P10-06).
func severeEvents() []severeEvent {
	return []severeEvent{
		{"Tornado Warning", SevRed},
		{"Extreme Wind Warning", SevRed},
		{"Hurricane Warning", SevRed},
		{"Severe Thunderstorm Warning", SevOrange},
		{"Flash Flood Warning", SevOrange},
		{"Tornado Watch", SevYellow},
		{"Hurricane Watch", SevYellow},
		{"Severe Thunderstorm Watch", SevYellow},
	}
}

// NWS reads the national active-alerts feed filtered to the severe/tornado
// event list (US only, keyless).
type NWS struct {
	client *httpx.Client
	base   string
}

// NewNWS builds the source; base "" is the production active-alerts endpoint.
func NewNWS(client *httpx.Client, base string) *NWS {
	if base == "" {
		base = "https://api.weather.gov/alerts/active"
	}
	return &NWS{client: client, base: base}
}

func (n *NWS) Name() string { return "NWS" }

func (n *NWS) url() string {
	list := severeEvents()
	events := make([]string, len(list))
	for i, e := range list {
		events[i] = e.event
	}
	q := url.Values{}
	q.Set("event", strings.Join(events, ",")) // one comma-joined param — NWS 400s on repeated event=, and on event+limit together (probed 2026-08-27)
	q.Set("status", "actual")
	// The event filter already bounds the result (the stack caps client-side);
	// NWS requires %20 for spaces, not the '+' url.Values emits.
	return n.base + "?" + strings.ReplaceAll(q.Encode(), "+", "%20")
}

type nwsFeed struct {
	Features []struct {
		ID         string `json:"id"` // the alert URI — stable
		Properties struct {
			Event      string `json:"event"`
			AreaDesc   string `json:"areaDesc"`
			Onset      string `json:"onset"`
			Sent       string `json:"sent"`
			Ends       string `json:"ends"`    // when the hazard ends (may be empty)
			Expires    string `json:"expires"` // when the alert message expires — always present
			References []struct {
				ID string `json:"@id"` // a prior alert this message updates/replaces
			} `json:"references"`
		} `json:"properties"`
		Geometry struct {
			Coordinates json.RawMessage `json:"coordinates"` // GeoJSON Point/Polygon/MultiPolygon, or absent (zone-only)
		} `json:"geometry"`
	} `json:"features"`
}

// geoPoint returns a representative point (the first vertex) of a GeoJSON
// coordinates blob — Point, Polygon or MultiPolygon — for the distance filter.
// A polygon's first vertex is close enough to its centre for a miles-scale
// proximity test; an absent/empty geometry yields ok=false (a zone-only alert,
// filtered out when a radius is set).
//
// It streams tokens rather than json.Unmarshal-ing into `any`: the interface
// decode path recurses per array level with no depth cap, so a hostile
// deeply-nested `coordinates` would overflow the stack and crash the process
// (red-team 0.12.0 P4 F2). Token() is iterative; the first vertex's lon,lat are
// the first two numbers, reached within a handful of tokens.
func geoPoint(raw json.RawMessage) (lat, lon float64, ok bool) {
	if len(raw) == 0 {
		return 0, 0, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	var nums []float64
	for i := 0; i < 64 && len(nums) < 2; i++ { // the first coordinate pair is a few open-brackets deep
		tok, err := dec.Token()
		if err != nil {
			return 0, 0, false
		}
		if f, isNum := tok.(float64); isNum {
			nums = append(nums, f)
		}
	}
	if len(nums) < 2 {
		return 0, 0, false
	}
	return nums[1], nums[0], true // GeoJSON is [lon, lat]
}

func (n *NWS) Fetch(ctx context.Context) ([]Event, error) {
	var feed nwsFeed
	if _, err := n.client.GetJSON(ctx, n.url(), &feed, httpx.TTL(nwsTTL)); err != nil {
		return nil, err
	}
	// NWS issues a NEW id when it updates/replaces a warning, linking the prior
	// via `references`. If both are briefly active, drop the superseded one so
	// the same real-world warning isn't shown (or announced) twice (0.12.0
	// follow-up — dedup beyond source id).
	superseded := make(map[string]bool)
	for _, f := range feed.Features {
		for _, r := range f.Properties.References {
			if r.ID != "" && r.ID != f.ID { // never let an alert supersede itself (guard a self-reference)
				superseded[r.ID] = true
			}
		}
	}
	out := make([]Event, 0, len(feed.Features))
	for _, f := range feed.Features {
		if f.ID == "" || f.Properties.Event == "" {
			continue
		}
		when := f.Properties.Onset
		if when == "" {
			when = f.Properties.Sent
		}
		at, _ := time.Parse(time.RFC3339, when)
		if at.IsZero() {
			at = time.Now() // a malformed/absent time is "as of now", not year 1 (else it mis-sorts and narrates a bogus clock — P4 F6)
		}
		// The active window ends at `ends` (the hazard) when given, else
		// `expires` (the message) — the marquee drops the alert past it (#2).
		until := f.Properties.Ends
		if until == "" {
			until = f.Properties.Expires
		}
		ends, _ := time.Parse(time.RFC3339, until)
		lat, lon, ok := geoPoint(f.Geometry.Coordinates) // ok=false for a zone-only alert (no geometry) → excluded when a radius is set
		out = append(out, Event{
			ID:         f.ID,
			Class:      ClassSevereWx,
			Severity:   severeSeverity(f.Properties.Event),
			Type:       clampField(f.Properties.Event),    // the spoken type ("Tornado Warning"); bounded so a hostile feed can't blow up the narration render (P4 F5)
			Place:      clampField(f.Properties.AreaDesc), // the location; bounded likewise
			Lat:        lat,
			Lon:        lon,
			HasPoint:   ok,
			Superseded: superseded[f.ID], // a superseded alert rides along so the ticker seen-marks it, but is not displayed/announced
			At:         at.UTC(),
			Until:      ends.UTC(),
			Source:     n.Name(),
		})
	}
	return out, nil
}

// severeSeverity maps a curated event name to its tier (unknown ⇒ orange).
func severeSeverity(event string) Severity {
	for _, e := range severeEvents() {
		if e.event == event {
			return e.sev
		}
	}
	return SevOrange
}
