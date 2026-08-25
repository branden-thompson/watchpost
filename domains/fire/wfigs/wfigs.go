// Package wfigs reads NIFC's Wildland Fire Interagency Geospatial Services
// current-incident layer (B5, live-probed 2026-08-25; keyless, public
// domain): one GeoJSON query for every active wildfire in the country
// (~600, under the layer's 2,000-record cap), answered for each location by
// distance. It gives the fire a NAME, acres and containment — the words a
// person uses — where the satellites give heat.
package wfigs

import (
	"context"
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

// Provider is the WFIGS snapshot provider.
type Provider struct {
	client *httpx.Client
	base   string
	rules  fire.Rules
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
	var fc featureCollection
	if _, err := p.client.GetJSON(ctx, p.base+"?"+q.Encode(), &fc, httpx.TTL(layerTTL)); err != nil {
		frag.Err = fmt.Errorf("wfigs: %w", err)
		return frag, nil
	}
	for _, ref := range req.Locations {
		var ins []snapshot.Incident
		for _, f := range fc.Features {
			if len(f.Geometry.Coordinates) < 2 || f.Properties.Name == "" {
				continue
			}
			lon, lat := f.Geometry.Coordinates[0], f.Geometry.Coordinates[1]
			km, ok := fire.Near(ref, lat, lon, p.rules.IncidentRadiusKm)
			if !ok {
				continue
			}
			d := km
			in := snapshot.Incident{Name: f.Properties.Name, Lat: lat, Lon: lon, PercentContained: f.Properties.Contained, Acres: firstOf(f.Properties.Size, f.Properties.Final, f.Properties.Discovery, f.Properties.Initial), State: f.Properties.State,
				Source: snapshot.SourceInfo{Provider: p.ID(), DistanceKm: &d, IssuedAt: frag.FetchedAt}}
			if f.Properties.Discovered != nil {
				in.Discovered = time.UnixMilli(int64(*f.Properties.Discovered)).UTC()
			}
			ins = append(ins, in)
		}
		sort.SliceStable(ins, func(i, j int) bool { return acres(ins[i]) > acres(ins[j]) }) // the big ones first; dispatch-only records last
		if len(ins) > maxIncidents {
			ins = ins[:maxIncidents]
		}
		frag.PerLocation[snapshot.Key(ref)] = snapshot.PartialData{Fire: &snapshot.FireState{AsOf: frag.FetchedAt, Incidents: ins}}
	}
	return frag, nil
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
