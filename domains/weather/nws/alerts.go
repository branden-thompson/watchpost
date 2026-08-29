package nws

// alerts.go — active alerts by zone and their mapping onto locations. Split from provider.go by the quality pass (Q2, pure move).

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

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
	maxFieldRunes = 120
	maxIDRunes    = 200 // an id: the URL form of an OID is 31 runes longer (R5-B-05)
	maxProseRunes = 4000
	maxListLen    = 50
)

func clampRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func clampList(s []string) []string {
	if len(s) > maxListLen {
		s = s[:maxListLen]
	}
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = clampRunes(v, maxFieldRunes)
	}
	return out
}

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
	var payload struct {
		Features []struct {
			Properties alertProps `json:"properties"`
		} `json:"features"`
	}
	u := fmt.Sprintf("%s/alerts/active?status=actual&zone=%s", p.base, strings.Join(zones, ","))
	if _, err := p.client.GetJSON(ctx, u, &payload); err != nil {
		return fmt.Errorf("alerts: %w", err)
	}
	perKey := map[snapshot.LocationKey][]snapshot.Alert{}
	for _, f := range payload.Features {
		mapAlert(f.Properties, zoneToKeys, perKey)
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
func mapAlert(pr alertProps, zoneToKeys map[string][]snapshot.LocationKey, perKey map[snapshot.LocationKey][]snapshot.Alert) {
	if err := invariant.Check(pr.ID != "", "CAP alert without an id cannot be deduplicated"); err != nil {
		pr.ID = "no-id:" + pr.Headline // never drop an alert silently (RS-10)
	}
	a := snapshot.Alert{
		ID:          clampRunes(pr.ID, maxIDRunes), // the bare OID; the feed path bounds its URL form the same (R5-B-05)
		Event:       clampRunes(pr.Event, maxFieldRunes),
		Severity:    strings.ToLower(clampRunes(pr.Severity, maxFieldRunes)),
		Urgency:     strings.ToLower(clampRunes(pr.Urgency, maxFieldRunes)),
		Certainty:   strings.ToLower(clampRunes(pr.Certainty, maxFieldRunes)),
		MessageType: strings.ToLower(clampRunes(pr.MessageType, maxFieldRunes)),
		Sent:        pr.Sent, Effective: pr.Effective, Onset: pr.Onset,
		Expires: pr.Expires, Ends: pr.Ends,
		AreaDesc: clampRunes(pr.AreaDesc, maxFieldRunes), Headline: clampRunes(pr.Headline, maxFieldRunes),
		Description: clampRunes(pr.Description, maxProseRunes), Instruction: clampRunes(pr.Instruction, maxProseRunes),
		SenderName: clampRunes(pr.SenderName, maxFieldRunes),
		Source:     snapshot.SourceInfo{Provider: "nws", IssuedAt: pr.Sent},
	}
	var refs []string
	for _, r := range pr.References {
		refs = append(refs, r.ID)
	}
	a.References = clampList(refs)
	// Two passes (B1 red-team #1): the full zone list must be complete BEFORE
	// any location receives its copy, or early-matched locations get a
	// truncated CAP AffectedZones. The match runs over the FULL list — a
	// Winter Storm Warning can span 80 zones and a tracked location's zone may
	// be the 60th (0.13.0 red-team R3-A-01: bounding the retained copy must
	// never drop the alert); only the copy kept on the record is bounded.
	var zones []string
	for _, zURL := range pr.AffectedZones {
		zones = append(zones, lastSegment(zURL))
	}
	a.AffectedZones = clampList(zones)
	matched := map[snapshot.LocationKey]bool{}
	for _, z := range zones {
		for _, k := range zoneToKeys[z] {
			if !matched[k] {
				matched[k] = true
				perKey[k] = append(perKey[k], a)
			}
		}
	}
}
