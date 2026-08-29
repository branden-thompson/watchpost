package globalfeed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	memo   sourceMemo
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

// nwsProps is one CAP feature's properties (the fields the window reads).
type nwsProps struct {
	Event         string   `json:"event"`
	AreaDesc      string   `json:"areaDesc"`
	Headline      string   `json:"headline"`
	Description   string   `json:"description"`
	Instruction   string   `json:"instruction"`
	Severity      string   `json:"severity"`
	Certainty     string   `json:"certainty"`
	Urgency       string   `json:"urgency"`
	MessageType   string   `json:"messageType"`
	Category      string   `json:"category"`
	Response      string   `json:"response"`
	Sender        string   `json:"sender"`
	SenderName    string   `json:"senderName"`
	Onset         string   `json:"onset"`
	Sent          string   `json:"sent"`
	Effective     string   `json:"effective"`
	Ends          string   `json:"ends"`    // when the hazard ends (may be empty)
	Expires       string   `json:"expires"` // when the alert message expires — always present
	AffectedZones []string `json:"affectedZones"`
	References    []struct {
		ID string `json:"@id"` // a prior alert this message updates/replaces
	} `json:"references"`
	Parameters map[string][]string `json:"parameters"` // read through the allowlist only (S7)
}

// nwsFeature is one entry of the feed.
type nwsFeature struct {
	ID         string   `json:"id"` // the alert URI — stable
	Properties nwsProps `json:"properties"`
	Geometry   struct {
		Coordinates json.RawMessage `json:"coordinates"` // GeoJSON Point/Polygon/MultiPolygon, or absent (zone-only)
	} `json:"geometry"`
}

type nwsFeed struct {
	Features []json.RawMessage `json:"features"` // decoded one by one: a malformed value skips ITS entry, never the source (REVIEW R5-C-12)
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

// parseCAPTime reads an RFC3339 field; zero when absent or malformed.
func parseCAPTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t.UTC()
}

// firstParam reads one allowlisted CAP parameter (the first value), bounded.
func firstParam(params map[string][]string, key string) string {
	if v, ok := params[key]; ok && len(v) > 0 {
		return clampField(v[0])
	}
	return ""
}

// severeDetailOf bounds one CAP feature's record fields (P4 F5 / NFR-5): short
// fields to maxFieldRunes, prose to maxProseRunes, lists to maxListLen, and the
// parameters map through the allowlist only (S7).
func severeDetailOf(p nwsProps) *SevereDetail {
	d := &SevereDetail{
		Headline: clampField(p.Headline), Description: clampProse(p.Description), Instruction: clampProse(p.Instruction),
		Severity: clampField(p.Severity), Certainty: clampField(p.Certainty), Urgency: clampField(p.Urgency),
		MessageType: clampField(p.MessageType), Category: clampField(p.Category), Response: clampField(p.Response),
		SenderName: clampField(p.SenderName), Sender: clampField(p.Sender),
		Effective: parseCAPTime(p.Effective), Sent: parseCAPTime(p.Sent), Expires: parseCAPTime(p.Expires),
		Ends: parseCAPTime(p.Ends), Onset: parseCAPTime(p.Onset),
		AffectedZones: clampSlice(p.AffectedZones),
		MaxWindGust:   firstParam(p.Parameters, "maxWindGust"), MaxHailSize: firstParam(p.Parameters, "maxHailSize"),
		EventMotion: firstParam(p.Parameters, "eventMotionDescription"), NWSHeadline: firstParam(p.Parameters, "NWSheadline"),
		VTEC: firstParam(p.Parameters, "VTEC"),
	}
	refs := make([]string, 0, len(p.References))
	for _, x := range p.References {
		refs = append(refs, x.ID)
	}
	d.References = clampSlice(refs)
	return d
}

func (n *NWS) Fetch(ctx context.Context) ([]Event, error) {
	u := n.url()
	return n.memo.events(func() ([]byte, bool, error) {
		var body []byte
		hdr, err := n.client.GetJSON(ctx, u, &body, httpx.TTL(nwsTTL))
		return body, hdr == nil, err
	}, func(body []byte) ([]Event, error) { return n.parse(body, u) })
}

func (n *NWS) parse(body []byte, u string) ([]Event, error) {
	var feed nwsFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		if n.client != nil {
			n.client.Forget(u)
		}
		return nil, fmt.Errorf("NWS: bad response body: %w", err)
	}
	feats := decodeFeatures(feed.Features)
	// NWS issues a NEW id when it updates/replaces a warning, linking the prior
	// via `references`. The guarded rule (supersede.go) drops the prior only
	// when the update is a newer message from the same sender for the same
	// product — never on a bare reference (red-team S3). SenderName keys the
	// sender on BOTH paths (the location path has no `sender` email).
	refs := make([]Ref, 0, len(feed.Features))
	for _, f := range feats {
		r := Ref{ID: f.ID, Sender: f.Properties.SenderName, Product: f.Properties.Event, Sent: parseCAPTime(f.Properties.Sent)}
		for _, x := range f.Properties.References {
			if x.ID != "" && x.ID != f.ID {
				r.Replaces = append(r.Replaces, x.ID)
			}
		}
		refs = append(refs, r)
	}
	superseded := SupersededBy(refs)
	out := make([]Event, 0, len(feed.Features))
	for _, f := range feats {
		p := f.Properties
		if f.ID == "" || p.Event == "" {
			continue
		}
		// "Declared" is the ISSUE time (sent), not the onset: a heat advisory
		// issued at 09:00 for 20:00 is declared at 09:00 and starts at 20:00
		// (HUM LEAD UAT 2026-08-28 — a future "declared" read as bad data).
		// The onset rides in the detail as "Starts".
		when := p.Sent
		if when == "" {
			when = p.Effective
		}
		if when == "" {
			when = p.Onset
		}
		at, _ := time.Parse(time.RFC3339, when)
		if at.IsZero() {
			at = time.Now() // a malformed/absent time is "as of now", not year 1 (else it mis-sorts and narrates a bogus clock — P4 F6)
		}
		// The active window ends at `ends` (the hazard) when given, else
		// `expires` (the message) — the marquee drops the alert past it (#2).
		until := p.Ends
		if until == "" {
			until = p.Expires
		}
		ends, _ := time.Parse(time.RFC3339, until)
		lat, lon, ok := geoPoint(f.Geometry.Coordinates) // ok=false for a zone-only alert (no geometry) → excluded when a radius is set
		out = append(out, Event{
			ID:         clampID(f.ID), // bounded (R3-D-02; ids get the longer bound, R5-B-05); the supersede map is keyed on the raw id above, looked up with the same
			Class:      ClassSevereWx,
			Severity:   severeSeverity(p.Event),
			Type:       clampField(p.Event),    // the spoken type ("Tornado Warning"); bounded so a hostile feed can't blow up the narration render (P4 F5)
			Place:      clampField(p.AreaDesc), // the location; bounded likewise
			Lat:        lat,
			Lon:        lon,
			HasPoint:   ok,
			Superseded: superseded[f.ID], // a superseded alert rides along so the ticker seen-marks it, but is not displayed/announced
			At:         at.UTC(),
			Until:      ends.UTC(),
			Source:     n.Name(),
			Severe:     severeDetailOf(p),
		})
	}
	return out, nil
}

// CuratedSeverity is the tier of a product on the curated national list —
// the one authority for it by every path (domains/severe reads it too).
func CuratedSeverity(product string) (Severity, bool) {
	for _, e := range severeEvents() {
		if e.event == product {
			return e.sev, true
		}
	}
	return SevOrange, false
}

// severeSeverity maps a curated event name to its tier (unknown ⇒ orange).
func severeSeverity(event string) Severity {
	sev, _ := CuratedSeverity(event)
	return sev
}

// decodeFeatures decodes the feed's entries one by one: a malformed entry
// skips itself, never the source (REVIEW R5-C-12).
func decodeFeatures(raw []json.RawMessage) []nwsFeature {
	feats := make([]nwsFeature, 0, len(raw))
	for _, r := range raw {
		var f nwsFeature
		if json.Unmarshal(r, &f) == nil {
			feats = append(feats, f)
		}
	}
	return feats
}
