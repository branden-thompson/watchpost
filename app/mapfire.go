package app

// mapfire.go — the map's fire (0.18.0 D-121): the active perimeters, each
// named incident, the satellite hotspots - one layer, on by default. The
// incidents and hotspots are what the places' fire already reads, nationally
// and memoised; only the perimeters are asked for here, a field box at a
// time and generalised to the map's resolution.

import (
	"context"
	"fmt"
	"strconv"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/domains/fire/hms"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Fire, registered: on by default, as alerts and quakes are (D-76, D-121).
func init() {
	registerMapLayer(mapLayer{key: tty.FireLayer, label: "Fire", on: true, cost: fireLayerCost, chips: []string{"NIFC", "HMS"}})
}

// fireInView is the fire the feed draws: the perimeters of the view's field
// boxes, and every incident and hotspot in the country, cut to the view
// when drawn.
type fireInView struct {
	perimeters []wfigs.Perimeter
	incidents  []snapshot.Incident
	hotspots   []hms.Point
}

// fireKeeps is how long fire is drawn as current: its feeds refresh every
// ten minutes, and a refresh that drops a fire takes it off the map.
const fireKeeps = time.Hour

// hmsCredit is the hotspots' credit line.
const hmsCredit = "NOAA HMS satellite fire detections"

// fireRules are the places' fire rules (config's [fire]), or their defaults.
func (lp *livePipelines) fireRules() fire.Rules {
	if lp == nil || lp.rules.RadiusKm == 0 {
		return fire.DefaultRules()
	}
	return lp.rules
}

// fireIn fetches the fire for an ask: the places' own providers, found
// among the fire set (app/fire.go); each that does not answer is simply
// not drawn.
func (lp *livePipelines) fireIn(ctx context.Context, ask tty.MapAsk) fireInView {
	var out fireInView
	for _, p := range lp.fire {
		switch p := p.(type) {
		case *hms.Provider:
			out.hotspots, _ = p.Points(ctx)
		case *wfigs.Provider:
			out.incidents, _ = p.Incidents(ctx)
			seen := map[string]bool{}
			for _, b := range fieldBoxes(ask.Region, ask.View) {
				ps, err := p.Perimeters(ctx, b.W, b.S, b.E, b.N)
				if err != nil {
					continue
				}
				for _, pm := range ps {
					if k := perimeterKey(pm); !seen[k] {
						seen[k] = true // a perimeter across two boxes is in both answers
						out.perimeters = append(out.perimeters, pm)
					}
				}
			}
		}
	}
	return out
}

// perimeterKey tells perimeters apart: a name and where its first outline
// begins - two fires may share a name.
func perimeterKey(p wfigs.Perimeter) string {
	if len(p.Areas) == 0 || len(p.Areas[0]) == 0 || len(p.Areas[0][0]) == 0 {
		return p.Name
	}
	at := p.Areas[0][0][0]
	return fmt.Sprintf("%s@%.3f,%.3f", p.Name, at[0], at[1])
}

// fireOverlays is the fire in view as the library's features, in fire's own
// roles (go-tuiMaps L-18): each perimeter its outlines and holes, unlabelled
// - its incident carries the words; each incident a marker labelled with its
// name, acres and containment; the hotspots the rules keep, one overlay of
// dots, the strong in fire's role and the rest fainter.
func fireOverlays(f fireInView, view tty.MapView, rules fire.Rules) []tuimaps.Overlay {
	now := time.Now()
	var out []tuimaps.Overlay
	var perims []tuimaps.Feature // one overlay a kind, never a fire (D-129, UAT-2 U2-34)
	for _, p := range f.perimeters {
		for _, area := range p.Areas {
			if !areaMeets(area, view) {
				continue
			}
			var rings [][]tuimaps.LonLat
			for _, r := range area {
				ring := make([]tuimaps.LonLat, len(r))
				for i, pt := range r {
					ring[i] = tuimaps.LonLat{Lon: pt[0], Lat: pt[1]}
				}
				rings = append(rings, ring)
			}
			perims = append(perims, tuimaps.Feature{Kind: tuimaps.Polygon, Rings: rings, Role: tuimaps.Fire, ID: perimeterKey(p)})
		}
	}
	if len(perims) > 0 {
		out = append(out, tuimaps.Overlay{ID: tty.FireLayer + "/perimeters", Valid: now, Keeps: fireKeeps, Credit: wfigs.Attribution, Features: perims})
	}
	var incidents []tuimaps.Feature
	for _, in := range f.incidents {
		if !view.Contains(in.Lat, in.Lon) {
			continue
		}
		incidents = append(incidents, tuimaps.Feature{Kind: tuimaps.Point, Rings: [][]tuimaps.LonLat{{{Lon: in.Lon, Lat: in.Lat}}},
			Role: tuimaps.Fire, Label: incidentLabel(in), ID: in.Name + "@" + strconv.FormatFloat(in.Lat, 'f', 3, 64)})
	}
	if len(incidents) > 0 {
		out = append(out, tuimaps.Overlay{ID: tty.FireLayer + "/incidents", Valid: now, Keeps: fireKeeps, Credit: wfigs.Attribution, Features: incidents})
	}
	var dots []tuimaps.Feature
	for _, h := range f.hotspots {
		if !view.Contains(h.Lat, h.Lon) || !rules.Keep("analyst", h.FRPMW) {
			continue
		}
		role := tuimaps.FireFaint
		if rules.Bold(snapshot.Hotspot{FRPMW: h.FRPMW}) {
			role = tuimaps.Fire // the stronger brighter (D-121)
		}
		dots = append(dots, tuimaps.Feature{Kind: tuimaps.Point, Rings: [][]tuimaps.LonLat{{{Lon: h.Lon, Lat: h.Lat}}}, Role: role})
	}
	if len(dots) > 0 {
		out = append(out, tuimaps.Overlay{ID: tty.FireLayer + "/hotspots", Valid: now, Keeps: fireKeeps, Credit: hmsCredit, Features: dots})
	}
	return out
}

// areaMeets reports whether an area's outline has a point in the view, or
// the view lies inside it: a perimeter wider than the view still meets it.
func areaMeets(area [][][2]float64, v tty.MapView) bool {
	if len(area) == 0 {
		return false
	}
	w, s, e, n := 180.0, 90.0, -180.0, -90.0
	for _, pt := range area[0] {
		w, e, s, n = min(w, pt[0]), max(e, pt[0]), min(s, pt[1]), max(n, pt[1])
	}
	return w <= v.E && e >= v.W && s <= v.N && n >= v.S
}

// incidentLabel is an incident's words on the map: its name, its acres and
// how much is contained, where known - "Timber 12,915 ac, 26%".
func incidentLabel(in snapshot.Incident) string {
	label := in.Name
	if in.Acres != nil {
		label += " " + render.Thousands(*in.Acres) + " ac"
	}
	if in.PercentContained != nil {
		label += ", " + strconv.Itoa(int(*in.PercentContained+0.5)) + "%"
	}
	return label
}

// fireHosts are fire's entries for the Status window's MAP block.
func fireHosts() []tty.MapSource {
	return []tty.MapSource{
		{Name: "NIFC WFIGS", Host: hostOf(wfigs.New(nil, "", fire.DefaultRules()).PerimetersBase()), Layers: "fire"},
		{Name: "NOAA HMS", Host: hostOf(hms.DefaultURL), Layers: "fire"},
	}
}

// perimeterBytes is the lower 48's perimeters on the wire, measured: 378 KB
// for its whole box at a hundredth of a degree (2026-09-27). A box is
// charged its share of that width.
const perimeterBytes = 378_000

// fireLayerCost is what fire would fetch in a refresh as if nothing were
// held: a perimeter request a field box. The incidents and the hotspots are
// the places' own reads, charged there.
func fireLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	boxes := fieldBoxes(in.region, in.view)
	total := int64(0)
	for _, b := range boxes {
		total += int64(float64(perimeterBytes) * min((b.E-b.W)/61, 1))
	}
	return total, len(boxes)
}

// fireChosen is the fire as the Fire row chooses (D-145): all of it; the
// named fires alone - perimeters and incidents; or the satellite hotspots
// alone.
func fireChosen(overlays []tuimaps.Overlay, mode string) []tuimaps.Overlay {
	if mode != tty.FireNamed && mode != tty.FireHotspots {
		return overlays
	}
	var out []tuimaps.Overlay
	for _, o := range overlays {
		if (o.ID == tty.FireLayer+"/hotspots") == (mode == tty.FireHotspots) {
			out = append(out, o)
		}
	}
	return out
}
