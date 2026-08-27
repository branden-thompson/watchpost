// Package wfigs reads NIFC's Wildland Fire Interagency Geospatial Services
// current-incident layer (B5, live-probed 2026-08-25; keyless, public
// domain): one GeoJSON query for every active wildfire in the country
// (~600, under the layer's 2,000-record cap), answered for each location by
// distance. It gives the fire a NAME, acres and containment — the words a
// person uses — where the satellites give heat.
package wfigs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Attribution is the credit line (NIFC Open Data, public domain).
const Attribution = "NIFC WFIGS wildfire incidents (nifc.gov)"

// layerTTL: incidents sync continuously; ten minutes matches the fire tier.
const layerTTL = 10 * time.Minute

// maxIncidents caps what a location lists (nearest, largest first).
const maxIncidents = 5

// Provider is the WFIGS snapshot provider. The decoded layer is memoised
// by body hash (quality pass Q3, L4-F6): every RECENT location's scheduler
// asks on its own tick, and the 208 KB layer decoded ~200 times an hour
// (~57 MB/h of garbage) for the same bytes; now once per change.
type Provider struct {
	client *httpx.Client
	base   string
	rules  fire.Rules
	memo   fire.Memo[[]incident]
}

// MemoIncidents reports how many decoded incidents the layer memo holds
// (the diagnostic dump's view of the memo).
func (p *Provider) MemoIncidents() int {
	ins, _ := p.memo.Peek()
	return len(ins)
}

// MemoStats is the memo's size and its decode count since launch.
func (p *Provider) MemoStats() (incidents, parses int) { return p.MemoIncidents(), p.memo.Parses() }

// incident is the layer's record in the shape Fetch needs: decoded once,
// answered for every location by distance.
type incident struct {
	lat, lon   float64
	name       string
	state      string
	contained  *float64
	acres      *float64
	discovered time.Time
}

// New builds the provider; base "" means the production layer.
func New(client *httpx.Client, base string, rules fire.Rules) *Provider {
	if base == "" {
		base = "https://services3.arcgis.com/T4QMspbfLg3qTGWY/arcgis/rest/services/WFIGS_Incident_Locations_Current/FeatureServer/0/query"
	}
	return &Provider{client: client, base: base, rules: rules}
}

// ID implements snapshot.Provider.
func (p *Provider) ID() string { return "wfigs" }

// Domains implements snapshot.Provider.
func (p *Provider) Domains() []string { return []string{"fire"} }

type featureCollection struct {
	Features []struct {
		Geometry struct {
			Coordinates []float64 `json:"coordinates"`
		} `json:"geometry"`
		Properties struct {
			Name       string   `json:"IncidentName"`
			Discovered *float64 `json:"FireDiscoveryDateTime"` // epoch ms
			Contained  *float64 `json:"PercentContained"`
			Size       *float64 `json:"IncidentSize"`
			Final      *float64 `json:"FinalAcres"`           // acreage fallbacks (live 2026-08-25: young incidents carry
			Discovery  *float64 `json:"DiscoveryAcres"`       // no IncidentSize yet — the first size reported is better
			Initial    *float64 `json:"InitialResponseAcres"` // than "n/a")
			State      string   `json:"POOState"`
			Category   string   `json:"IncidentTypeCategory"`
		} `json:"properties"`
	} `json:"features"`
}

// Fetch implements snapshot.Provider for KindFire.
func (p *Provider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	frag := snapshot.Fragment{Provider: p.ID(), Kind: req.Kind, FetchedAt: time.Now().UTC(), PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	if err := invariant.Check(req.Kind == snapshot.KindFire, "wfigs serves only KindFire"); err != nil {
		return frag, err
	}
	if err := p.rules.Valid(); err != nil {
		return frag, err
	}
	q := url.Values{}
	q.Set("where", "IncidentTypeCategory='WF'")
	q.Set("outFields", "IncidentName,FireDiscoveryDateTime,PercentContained,IncidentSize,FinalAcres,DiscoveryAcres,InitialResponseAcres,POOState,IncidentTypeCategory")
	q.Set("outSR", "4326")
	q.Set("resultRecordCount", "2000")
	q.Set("f", "geojson")
	u := p.base + "?" + q.Encode()
	raw, err := p.client.GetText(ctx, u, httpx.TTL(layerTTL)) // read-only (httpx.GetText contract)
	if err != nil {
		frag.Err = fmt.Errorf("wfigs: %w", err)
		return frag, nil
	}
	layer, err := p.memo.Get(raw, decodeLayer)
	if err != nil {
		p.client.Forget(u) // a body that does not decode must not be served for the rest of its TTL
		frag.Err = fmt.Errorf("wfigs: %w", err)
		return frag, nil
	}
	for _, ref := range req.Locations {
		var ins []snapshot.Incident
		for _, f := range layer {
			km, ok := fire.Near(ref, f.lat, f.lon, p.rules.IncidentRadiusKm)
			if !ok {
				continue
			}
			d := km
			ins = append(ins, snapshot.Incident{Name: f.name, Lat: f.lat, Lon: f.lon, PercentContained: f.contained, Acres: f.acres, State: f.state, Discovered: f.discovered,
				Source: snapshot.SourceInfo{Provider: p.ID(), DistanceKm: &d, IssuedAt: frag.FetchedAt}})
		}
		sort.SliceStable(ins, func(i, j int) bool { return acres(ins[i]) > acres(ins[j]) }) // the big ones first; dispatch-only records last
		if len(ins) > maxIncidents {
			ins = ins[:maxIncidents]
		}
		frag.PerLocation[snapshot.Key(ref)] = snapshot.PartialData{Fire: &snapshot.FireState{AsOf: frag.FetchedAt, Incidents: ins}}
	}
	return frag, nil
}

// decodeLayer decodes the GeoJSON layer into the compact incident list:
// features without a point or a name are dropped here, once.
func decodeLayer(raw []byte) ([]incident, error) {
	var fc featureCollection
	if err := json.Unmarshal(raw, &fc); err != nil {
		return nil, fmt.Errorf("bad response body: %w", err)
	}
	out := make([]incident, 0, len(fc.Features))
	for _, f := range fc.Features {
		if len(f.Geometry.Coordinates) < 2 || f.Properties.Name == "" {
			continue
		}
		in := incident{lon: f.Geometry.Coordinates[0], lat: f.Geometry.Coordinates[1], name: f.Properties.Name, state: f.Properties.State,
			contained: f.Properties.Contained, acres: firstOf(f.Properties.Size, f.Properties.Final, f.Properties.Discovery, f.Properties.Initial)}
		if f.Properties.Discovered != nil {
			in.discovered = time.UnixMilli(int64(*f.Properties.Discovered)).UTC()
		}
		out = append(out, in)
	}
	return out, nil
}

func acres(in snapshot.Incident) float64 {
	if in.Acres == nil {
		return -1
	}
	return *in.Acres
}

// firstOf is the first reported acreage, nil when none is.
func firstOf(vs ...*float64) *float64 {
	for _, v := range vs {
		if v != nil {
			return v
		}
	}
	return nil
}
