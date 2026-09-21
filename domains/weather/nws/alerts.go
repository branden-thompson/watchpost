package nws

// alerts.go — active alerts by zone and their mapping onto locations. Split from provider.go by the quality pass (Q2, pure move).

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/plaintext"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

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
	SenderName    string   `json:"senderName"`
}

// Field bounds for a CAP alert reaching the snapshot (0.13.0, NFR-5; red-team
// S2 — the location path was unbounded while the ticker path was not).
const (
	maxIDRunes    = 200 // an id: the URL form of an OID is 31 runes longer (R5-B-05)
	maxProseRunes = 4000
	// maxZones bounds the zone ids kept on an alert. **The general list bound
	// is fifty, and these are not prose** - they are short identifiers, and
	// they are what the alert's ground is resolved from, so a clamp here is a
	// piece of the map quietly missing. A Winter Storm Warning can name eighty
	// zones; the most measured live was forty-two. This is well above both and
	// still bounded.
	maxZones = 256
)

func (p *Provider) fetchAlerts(ctx context.Context, refs []snapshot.LocationRef, frag *snapshot.Fragment) error {
	// Collect every location's zones (dual-UGC: forecastZone + county — M3).
	zoneToKeys := map[string][]snapshot.LocationKey{}
	var zones []string
	// **One place that cannot be resolved must not cost the others theirs.**
	// Returning here left EVERY watched location with no alerts, including
	// places that were perfectly reachable: one bad lookup and the whole
	// station went quiet. The error is kept and returned only if nothing
	// resolved at all, so a total failure still degrades loudly.
	var lastErr error
	for _, ref := range refs {
		g, err := p.resolve(ctx, ref)
		if err != nil {
			lastErr = err
			continue
		}
		k := snapshot.Key(ref)
		for _, z := range g.zones {
			if _, seen := zoneToKeys[z]; !seen {
				zones = append(zones, z)
			}
			zoneToKeys[z] = append(zoneToKeys[z], k)
		}
	}
	if len(zones) == 0 && lastErr != nil {
		return lastErr
	}
	sort.Strings(zones)
	var payload alertsPayload
	u := fmt.Sprintf("%s/alerts/active?status=actual&zone=%s", p.base, strings.Join(zones, ","))
	if _, err := p.client.GetJSON(ctx, u, &payload); err != nil {
		return fmt.Errorf("alerts: %w", err)
	}
	perKey := map[snapshot.LocationKey][]snapshot.Alert{}
	for _, f := range payload.Features {
		mapAlert(f.Properties, f.Geometry, zoneToKeys, perKey)
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

// alertsPayload is the shape of the service's answer. **It declares the
// geometry**, which is what keeps an alert's own polygon: a field `encoding/json`
// is not told about is discarded at decode with nothing recording that it was.
type alertsPayload struct {
	Features []alertFeature `json:"features"`
}

type alertFeature struct {
	Properties alertProps      `json:"properties"`
	Geometry   json.RawMessage `json:"geometry"`
}

// decodeAlerts reads the service's answer into alerts, without attaching them
// to any location. It exists so a test can drive the decode: a seam a test
// cannot reach is not a covered seam (P-1).
func decodeAlerts(body []byte) ([]snapshot.Alert, error) {
	var payload alertsPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	out := make([]snapshot.Alert, 0, len(payload.Features))
	for _, f := range payload.Features {
		out = append(out, alertFrom(f.Properties, f.Geometry))
	}
	return out, nil
}

// mapAlert converts one CAP feature and attaches it to every watched location
// whose zones it affects, once per location.
func mapAlert(pr alertProps, geom json.RawMessage, zoneToKeys map[string][]snapshot.LocationKey, perKey map[snapshot.LocationKey][]snapshot.Alert) {
	a := alertFrom(pr, geom) // the copy kept on the record, its lists bounded
	// **The match runs over the FULL zone list, not the bounded copy** (B1
	// red-team #1; 0.13.0 red-team R3-A-01). A Winter Storm Warning can span
	// eighty zones and a tracked location's may be the sixtieth, so matching
	// on `a.AffectedZones` - which alertFrom bounds - would drop the alert for
	// exactly the locations furthest down the list. The bound is far above any
	// alert measured now (maxZones), but the rule stands on its own: the match
	// reads what arrived, never the copy that was kept.
	matched := map[snapshot.LocationKey]bool{}
	for _, zURL := range pr.AffectedZones {
		for _, k := range zoneToKeys[lastSegment(zURL)] {
			if !matched[k] {
				matched[k] = true
				perKey[k] = append(perKey[k], a)
			}
		}
	}
}

// standInID is the identity for a CAP alert that arrived without one.
//
// **The headline alone is not an identity.** The service writes the same
// headline for every warning of a kind - "Tornado Warning issued" is what all
// of them say - so two live hazards collided on one id, and everything keyed by
// it kept one and lost the other: the ground resolved for drawing, the
// read-once mark, the dedupe.
//
// What separates two alerts is where and when: the area described, who issued
// it, and the minute it was sent. Hashed rather than concatenated so the result
// is a bounded id whatever arrives in those fields.
func standInID(pr alertProps) string {
	h := fnv.New64a()
	for _, part := range []string{pr.Headline, pr.Event, pr.AreaDesc, pr.SenderName, pr.Sent.UTC().Format(time.RFC3339)} {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0}) // so "ab"+"c" and "a"+"bc" are not one id
	}
	return fmt.Sprintf("no-id:%016x", h.Sum64())
}

// alertFrom turns one CAP feature into an alert, geometry included.
func alertFrom(pr alertProps, geom json.RawMessage) snapshot.Alert {
	if err := invariant.Check(pr.ID != "", "CAP alert without an id cannot be deduplicated"); err != nil {
		pr.ID = standInID(pr) // never drop an alert silently (RS-10)
	}
	a := snapshot.Alert{
		ID:          plaintext.ClampRunes(pr.ID, maxIDRunes), // the bare OID; the feed path bounds its URL form the same (R5-B-05)
		Event:       plaintext.ClampField(pr.Event),
		Severity:    strings.ToLower(plaintext.ClampField(pr.Severity)),
		Urgency:     strings.ToLower(plaintext.ClampField(pr.Urgency)),
		Certainty:   strings.ToLower(plaintext.ClampField(pr.Certainty)),
		MessageType: strings.ToLower(plaintext.ClampField(pr.MessageType)),
		Sent:        pr.Sent, Effective: pr.Effective, Onset: pr.Onset,
		Expires: pr.Expires, Ends: pr.Ends,
		AreaDesc: plaintext.ClampField(pr.AreaDesc), Headline: plaintext.ClampField(pr.Headline),
		Description: plaintext.ClampRunes(pr.Description, maxProseRunes), Instruction: plaintext.ClampRunes(pr.Instruction, maxProseRunes),
		SenderName: plaintext.ClampField(pr.SenderName),
		Source:     snapshot.SourceInfo{Provider: "nws", IssuedAt: pr.Sent},
	}
	area, err := geo.ReadGeometry(geom)
	if err != nil {
		area = nil // a shape we cannot read is no shape; the alert still stands
	}
	a.Area = area
	var refs []string
	for _, r := range pr.References {
		refs = append(refs, r.ID)
	}
	a.References = plaintext.ClampList(refs)
	var zones []string
	for _, zURL := range pr.AffectedZones {
		if len(zones) == maxZones {
			break
		}
		zones = append(zones, plaintext.ClampField(lastSegment(zURL)))
	}
	a.AffectedZones = zones
	return a
}
