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
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/bodymemo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
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
// asks on its own tick, and decoding the 208 KB layer ~200 times an hour
// would be ~57 MB/h of garbage for the same bytes; it decodes once per change.
type Provider struct {
	client     *httpx.Client
	base       string
	perimeters string // the interagency perimeters' layer, on the same host (0.18.0 D-121)
	rules      fire.Rules
	memo       *bodymemo.Memo[struct{}, []incident] // the layer's parse, errors kept (Q3)
	// perimeterMemo is each box's perimeters by its URL (W14, P-7): a box's
	// body (up to ~378 KB) is served from the cache on every map ask and decoded
	// once, not each time. A few boxes a view; eight kept.
	perimeterMemo *bodymemo.Memo[string, []Perimeter]
}

// newPerimeterMemo is the memo's constructor as a value: P10's call graph
// matches a call by its bare name, and bodymemo.New called inside this
// package's New reads as New calling itself (W14).
var newPerimeterMemo = bodymemo.New[string, []Perimeter]

// PerimeterParses is how many perimeter bodies have been decoded.
func (p *Provider) PerimeterParses() int {
	if p.perimeterMemo == nil {
		return 0
	}
	_, n := p.perimeterMemo.Stats()
	return n
}

// MemoIncidents reports how many decoded incidents the layer memo holds
// (the diagnostic dump's view of the memo).
func (p *Provider) MemoIncidents() int {
	ins, _ := p.memo.Last(struct{}{})
	return len(ins)
}

// MemoStats is the memo's size and its decode count since launch.
func (p *Provider) MemoStats() (incidents, parses int) {
	_, parses = p.memo.Stats()
	return p.MemoIncidents(), parses
}

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
	return &Provider{client: client, base: base, rules: rules, perimeterMemo: newPerimeterMemo(8), memo: bodymemo.NewKeepingErrors[struct{}, []incident](1),
		perimeters: strings.Replace(base, "WFIGS_Incident_Locations_Current", "WFIGS_Interagency_Perimeters_Current", 1)}
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
	layer, err := p.layer(ctx)
	if err != nil {
		frag.Err = err
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
		sort.SliceStable(ins, func(i, j int) bool { return ins[i].AcresOrMissing() > ins[j].AcresOrMissing() }) // the big ones first; dispatch-only records last
		if len(ins) > maxIncidents {
			ins = ins[:maxIncidents]
		}
		frag.PerLocation[snapshot.Key(ref)] = snapshot.PartialData{Fire: &snapshot.FireState{AsOf: frag.FetchedAt, IncidentsAsOf: frag.FetchedAt, Incidents: ins}}
	}
	return frag, nil
}

// layer is every active wildfire in the country: one query, its decode
// memoised by body.
func (p *Provider) layer(ctx context.Context) ([]incident, error) {
	q := url.Values{}
	q.Set("where", "IncidentTypeCategory='WF'")
	q.Set("outFields", "IncidentName,FireDiscoveryDateTime,PercentContained,IncidentSize,FinalAcres,DiscoveryAcres,InitialResponseAcres,POOState,IncidentTypeCategory")
	q.Set("outSR", "4326")
	q.Set("resultRecordCount", "2000")
	q.Set("f", "geojson")
	u := p.base + "?" + q.Encode()
	raw, err := p.client.GetText(ctx, u, httpx.TTL(layerTTL)) // read-only (httpx.GetText contract)
	if err != nil {
		return nil, fmt.Errorf("wfigs: %w", err)
	}
	layer, err := p.memo.Parsed(struct{}{}, raw, decodeLayer)
	if err != nil {
		p.client.Forget(u) // a body that does not decode must not be served for the rest of its TTL
		return nil, fmt.Errorf("wfigs: %w", err)
	}
	return layer, nil
}

// Incidents are every active wildfire in the country, for the map (0.18.0
// D-121): the places' own query and memo, none of the places' radius.
func (p *Provider) Incidents(ctx context.Context) ([]snapshot.Incident, error) {
	layer, err := p.layer(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]snapshot.Incident, 0, len(layer))
	for _, f := range layer {
		out = append(out, snapshot.Incident{Name: f.name, Lat: f.lat, Lon: f.lon, PercentContained: f.contained, Acres: f.acres, State: f.state, Discovered: f.discovered,
			Source: snapshot.SourceInfo{Provider: p.ID()}})
	}
	return out, nil
}

// PerimetersBase is the perimeters' layer: its host is on the Status
// window's list (FR-9.4).
func (p *Provider) PerimetersBase() string { return p.perimeters }

// Perimeter is one active fire's perimeter (0.18.0 D-121): its name, acres
// and containment, and its areas - each an outline, then its holes - as
// longitude, latitude.
type Perimeter struct {
	Name             string
	Acres, Contained *float64
	Areas            [][][][2]float64
}

// Perimeters are the active perimeters meeting a box, generalised to about
// a thousandth of its width - never finer than a thousandth of a degree nor
// coarser than a hundredth: the whole country's ungeneralised is 14.6 MB,
// the lower 48's box at a hundredth 378 KB (2026-09-27).
func (p *Provider) Perimeters(ctx context.Context, w, s, e, n float64) ([]Perimeter, error) {
	q := url.Values{}
	q.Set("where", "1=1")
	q.Set("outFields", "poly_IncidentName,poly_GISAcres,attr_PercentContained")
	q.Set("geometry", fmt.Sprintf("%g,%g,%g,%g", w, s, e, n))
	q.Set("geometryType", "esriGeometryEnvelope")
	q.Set("inSR", "4326")
	q.Set("outSR", "4326")
	q.Set("spatialRel", "esriSpatialRelIntersects")
	q.Set("maxAllowableOffset", strconv.FormatFloat(min(max((e-w)/5000, 0.001), 0.01), 'f', -1, 64))
	q.Set("geometryPrecision", "3")
	q.Set("f", "geojson")
	u := p.perimeters + "?" + q.Encode()
	raw, err := p.client.GetText(ctx, u, httpx.TTL(layerTTL))
	if err != nil {
		return nil, fmt.Errorf("wfigs perimeters: %w", err)
	}
	if p.perimeterMemo == nil {
		return decodePerimeters(raw)
	}
	out, err := p.perimeterMemo.Parsed(u, raw, decodePerimeters) // read-only to its callers: shared by the box's asks
	if err != nil {
		p.client.Forget(u) // a bad body is not served again
	}
	return out, err
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
			contained: f.Properties.Contained, acres: render.FirstOf(f.Properties.Size, f.Properties.Final, f.Properties.Discovery, f.Properties.Initial)}
		if f.Properties.Discovered != nil {
			in.discovered = time.UnixMilli(int64(*f.Properties.Discovered)).UTC()
		}
		out = append(out, in)
	}
	return out, nil
}

// firstOf is the first reported acreage, nil when none is.

// decodePerimeters is a perimeters body's decode: each active fire's name,
// acres, containment and areas. Top-level, so the memo's hits allocate nothing.
func decodePerimeters(raw []byte) ([]Perimeter, error) {
	var fc struct {
		Features []struct {
			Geometry *struct {
				Type        string          `json:"type"`
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				Name      string   `json:"poly_IncidentName"`
				Acres     *float64 `json:"poly_GISAcres"`
				Contained *float64 `json:"attr_PercentContained"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(raw, &fc); err != nil {
		return nil, fmt.Errorf("wfigs perimeters: bad response body: %w", err)
	}
	out := make([]Perimeter, 0, len(fc.Features))
	for _, f := range fc.Features {
		if f.Geometry == nil {
			continue
		}
		var areas [][][][2]float64
		switch f.Geometry.Type {
		case "Polygon":
			var one [][][2]float64
			if json.Unmarshal(f.Geometry.Coordinates, &one) == nil {
				areas = [][][][2]float64{one}
			}
		case "MultiPolygon":
			_ = json.Unmarshal(f.Geometry.Coordinates, &areas)
		}
		if len(areas) == 0 {
			continue
		}
		out = append(out, Perimeter{Name: f.Properties.Name, Acres: f.Properties.Acres, Contained: f.Properties.Contained, Areas: areas})
	}
	return out, nil
}
