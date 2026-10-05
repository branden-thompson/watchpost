package app

// mapinview.go — "Alerts in view", the map's default scope (0.18.0 D-66,
// UAT-1 U1-15).
//
// THE VIEW IS ASKED FOR BY AREA, NEVER BY RECTANGLE (D-47): the states and
// marine areas the view touches, in one request, remembered for two minutes
// - the ticker's own refresh - and kept to the view before they are drawn.
// The window asks only once the view has stood still (its settle tick), and
// the estimate, which runs on the UI goroutine, never fetches: it reads what
// was remembered.

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/agememo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// areaMemoAge is how long an answer for a set of areas is used again: the
// ticker's cadence, so the map is no staler than the marquee.
const areaMemoAge = 2 * time.Minute

// areaMemoKeys is how many sets of areas are remembered: a pan back over
// the last few views asks nothing again.
const areaMemoKeys = 8

// areaMemoRules are the area alerts' memo's: the last few sets of areas,
// each for areaMemoAge.
var areaMemoRules = agememo.Options{Fresh: areaMemoAge, Max: areaMemoKeys}

// areaFetchLimit bounds the shared ask for a set of areas' alerts. It runs
// under its own context, not the asker's: an ask a pan cancels would
// otherwise fail every ask that joined it (agememo shares one fetch).
const areaFetchLimit = 20 * time.Second

// viewAlerts is the alerts of the areas the view touches: remembered within
// areaMemoAge, else asked of the service - unless fetch is off, when only
// what was remembered is read, whatever its age (the estimate). A failed ask
// is the error, and the diagnostics are told (D-124); the station's own
// alerts still draw.
func (lp *livePipelines) viewAlerts(ctx context.Context, v tty.MapView, fetch bool) ([]snapshot.Alert, error) {
	areas := lp.areasOf.of(lp.idx, v)
	key := strings.Join(areas, ",")
	if !fetch {
		alerts, _, _ := lp.areaMemo.memo(areaMemoRules).Last(key)
		return alerts, nil
	}
	if lp.areaAlerts == nil || len(areas) == 0 {
		return nil, nil
	}
	alerts, err := lp.areaMemo.memo(areaMemoRules).Do(ctx, key, func() ([]snapshot.Alert, error) {
		own, cancel := context.WithTimeout(context.WithoutCancel(ctx), areaFetchLimit)
		defer cancel()
		return lp.areaAlerts(own, areas)
	})
	if err != nil {
		lp.problems.note("Alerts in view: the National Weather Service did not answer - " + errKind(err)) // the diagnostics', never the listener's (D-124)
		return nil, err
	}
	return alerts, nil
}

// errKind is an error said without its address: the kind of failure, which
// the diagnostics keep (the address may name places in view).
func errKind(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timed out"
	}
	var se *httpx.StatusError
	if errors.As(err, &se) {
		return "HTTP " + strconv.Itoa(se.Status)
	}
	return "no answer"
}

// inViewOnly keeps the alerts the view shows: a polygon that meets the view,
// and every alert that names zones (the areas asked for hold them; the map
// clips what lies outside).
func inViewOnly(alerts []snapshot.Alert, v tty.MapView) []snapshot.Alert {
	var out []snapshot.Alert
	for _, a := range alerts {
		if a.Area.Empty() || shapeMeets(a, v) {
			out = append(out, a)
		}
	}
	return out
}

// shapeMeets reports whether an alert's polygon's box meets the view.
func shapeMeets(a snapshot.Alert, v tty.MapView) bool {
	w, s, e, n := 180.0, 90.0, -180.0, -90.0
	for _, poly := range a.Area {
		for _, ring := range poly {
			for _, p := range ring {
				w, s, e, n = min(w, p.Lon), min(s, p.Lat), max(e, p.Lon), max(n, p.Lat)
			}
		}
	}
	return w <= v.E && e >= v.W && s <= v.N && n >= v.S
}

// viewSamples is how many points a side of the view is read at, for the
// areas it touches.
const viewSamples = 5

// lastViewAreas is the last view asked and its area codes, safe for the
// feed's goroutines and the window's at once.
type lastViewAreas struct {
	mu    sync.Mutex
	view  tty.MapView
	areas []string
	held  bool
}

// of is the view's area codes: the last answer when the view is the last
// one asked, else worked out and kept. The slice is shared: read-only.
func (l *lastViewAreas) of(idx *geodata.Index, v tty.MapView) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.held && l.view == v {
		return l.areas
	}
	l.view, l.areas, l.held = v, viewAreas(idx, v), true
	return l.areas
}

// viewAreas is the NWS area codes a view touches, sorted: each sample point's
// state, from the nearest town, or where there is none, the marine area it
// lies in.
func viewAreas(idx *geodata.Index, v tty.MapView) []string {
	seen := map[string]bool{}
	for i := range viewSamples {
		for j := range viewSamples {
			lat := v.S + (v.N-v.S)*float64(i)/float64(viewSamples-1)
			lon := v.W + (v.E-v.W)*float64(j)/float64(viewSamples-1)
			if code := stateNear(idx, lat, lon); code != "" {
				seen[code] = true
				continue
			}
			if code := marineArea(lat, lon); code != "" {
				seen[code] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// stateNear is the state of the nearest town within forty miles, or nothing
// over open water.
func stateNear(idx *geodata.Index, lat, lon float64) string {
	if idx == nil {
		return ""
	}
	near := idx.Near(lat, lon, 40, 1)
	if len(near) == 0 {
		return ""
	}
	return near[0].State
}

// marineArea is the NWS marine area a point over water lies in, coarsely:
// the Pacific coast, the Gulf, the Atlantic north and south of Hatteras, and
// the waters of Alaska, Hawaii, the Marianas, Samoa and Puerto Rico. The
// Great Lakes lie within forty miles of a town and read as their states.
func marineArea(lat, lon float64) string {
	switch {
	case lat > 50 && (lon < -129 || lon > 170):
		return "PK"
	case lat > 17 && lat < 24 && lon > -163 && lon < -153:
		return "PH"
	case lat > 12 && lat < 21.5 && lon > 143 && lon < 147:
		return "PM"
	case lat > -15.5 && lat < -10 && lon > -172 && lon < -167.5:
		return "PS"
	case lat > 16.5 && lat < 19.5 && lon > -68.5 && lon < -63.5:
		return "AM" // off Puerto Rico and the Virgin Islands: the AMZ7xx waters
	case lat > 30 && lat < 49.5 && lon > -130 && lon <= -117:
		return "PZ"
	case lat > 23 && lat < 31 && lon > -98 && lon < -81:
		return "GM"
	case lat > 24 && lat < 45 && lon >= -82 && lon < -64:
		if lat >= 35.2 {
			return "AN"
		}
		return "AM"
	}
	return ""
}
