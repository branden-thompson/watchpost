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
		ID:          pr.ID,
		Event:       pr.Event,
		Severity:    strings.ToLower(pr.Severity),
		Urgency:     strings.ToLower(pr.Urgency),
		Certainty:   strings.ToLower(pr.Certainty),
		MessageType: strings.ToLower(pr.MessageType),
		Sent:        pr.Sent, Effective: pr.Effective, Onset: pr.Onset,
		Expires: pr.Expires, Ends: pr.Ends,
		AreaDesc: pr.AreaDesc, Headline: pr.Headline,
		Description: pr.Description, Instruction: pr.Instruction,
		Source: snapshot.SourceInfo{Provider: "nws", IssuedAt: pr.Sent},
	}
	for _, r := range pr.References {
		a.References = append(a.References, r.ID)
	}
	// Two passes (B1 red-team #1): the full zone list must be complete BEFORE
	// any location receives its copy, or early-matched locations get a
	// truncated CAP AffectedZones.
	for _, zURL := range pr.AffectedZones {
		a.AffectedZones = append(a.AffectedZones, lastSegment(zURL))
	}
	matched := map[snapshot.LocationKey]bool{}
	for _, z := range a.AffectedZones {
		for _, k := range zoneToKeys[z] {
			if !matched[k] {
				matched[k] = true
				perKey[k] = append(perKey[k], a)
			}
		}
	}
}
