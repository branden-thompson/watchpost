package nws

// alerts.go — active alerts by zone and their mapping onto locations. Split from provider.go by the quality pass (Q2, pure move).

import (
	"context"
	"encoding/json"
	"fmt"
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
)

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

// mapAlert converts one CAP feature and attaches it to every watched location
// whose zones it affects (deduped per location).
// alertsPayload is the shape of the service's answer. **The geometry is
// declared** - before this it was not, so `encoding/json` discarded an alert's
// own polygon at decode with nothing recording that it did.
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

func mapAlert(pr alertProps, geom json.RawMessage, zoneToKeys map[string][]snapshot.LocationKey, perKey map[snapshot.LocationKey][]snapshot.Alert) {
	a := alertFrom(pr, geom) // the copy kept on the record, its lists bounded
	// **The match runs over the FULL zone list, not the bounded copy** (B1
	// red-team #1; 0.13.0 red-team R3-A-01). A Winter Storm Warning can span
	// eighty zones and a tracked location's may be the sixtieth, so matching
	// on `a.AffectedZones` - which alertFrom clamps - would drop the alert for
	// exactly the locations furthest down the list.
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

// alertFrom turns one CAP feature into an alert, geometry included.
func alertFrom(pr alertProps, geom json.RawMessage) snapshot.Alert {
	if err := invariant.Check(pr.ID != "", "CAP alert without an id cannot be deduplicated"); err != nil {
		pr.ID = "no-id:" + pr.Headline // never drop an alert silently (RS-10)
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
		zones = append(zones, lastSegment(zURL))
	}
	a.AffectedZones = plaintext.ClampList(zones)
	return a
}
