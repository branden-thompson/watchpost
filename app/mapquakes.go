package app

// mapquakes.go — the map's earthquakes (0.18.0 D-80, D-122, D-123): the
// quakes the listener chose from USGS's summary feeds - M2.5+ over the past
// week by default - drawn as USGS's map draws them: a ring around each
// epicentre, its size on the screen fixed by its magnitude, coloured by its
// age, labelled with its magnitude and local time. A layer of its own,
// switched in the Overlays menu.

import (
	"context"
	"math"
	"strconv"
	"sync"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/render"
)

// quakeLayerKey is the earthquakes' key in the registry, and their overlays'
// ids' first part.
const quakeLayerKey = "quake"

// The earthquakes, registered: on by default (D-80).
func init() {
	registerMapLayer(mapLayer{key: quakeLayerKey, label: "Earthquakes", on: true, cost: quakeLayerCost})
}

// quakeFeedBase is USGS's summary feeds' folder: each feed is its name and
// ".geojson" - on earthquake.usgs.gov, the ticker's and the places' host.
const quakeFeedBase = "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/"

// quakeFeedDefault is M2.5+ over the past week (D-122).
const quakeFeedDefault = "2.5_week"

// quakeFeedBytes are the feeds on the wire, measured (2026-09-27): the
// estimate's figure for each.
var quakeFeedBytes = map[string]int64{"2.5_day": 33_000, "2.5_week": 221_000, "1.0_day": 113_000, "1.0_week": 918_000}

// mapQuakes are the map's quake feeds, one source a feed, built as asked
// over the client they share.
type mapQuakes struct {
	client *httpx.Client
	base   string
	mu     sync.Mutex
	feeds  map[string]*globalfeed.USGS
}

// newMapQuakes builds the map's quake feeds over a client; base "" is USGS.
func newMapQuakes(c *httpx.Client, base string) *mapQuakes {
	if base == "" {
		base = quakeFeedBase
	}
	return &mapQuakes{client: c, base: base, feeds: map[string]*globalfeed.USGS{}}
}

// fetch is a feed's quakes, the default for a name not offered; none where
// it does not answer.
func (q *mapQuakes) fetch(ctx context.Context, name string) []globalfeed.Event {
	if q == nil {
		return nil
	}
	if _, ok := quakeFeedBytes[name]; !ok {
		name = quakeFeedDefault
	}
	q.mu.Lock()
	src, ok := q.feeds[name]
	if !ok {
		src = globalfeed.NewUSGS(q.client, q.base+name+".geojson")
		q.feeds[name] = src
	}
	q.mu.Unlock()
	events, err := src.Fetch(ctx)
	if err != nil {
		return nil
	}
	return events
}

// quakeLayerCost is the feed chosen, one request (D-122, D-23); nothing
// with no map region.
func quakeLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	if b, ok := quakeFeedBytes[in.quakeFeed]; ok {
		return b, 1
	}
	return quakeFeedBytes[quakeFeedDefault], 1
}

// quakeKeeps is how long a quake is drawn as current: the feeds are at most
// a week's, and the feed dropping it is what takes it off the map. SEVEN
// DAYS IS THE MOST THE LIBRARY KEEPS ANYTHING CURRENT: eight were refused,
// every quake of the week's feed a note of its own (UAT-2 U2-29).
const quakeKeeps = 7 * 24 * time.Hour

// quakesIn is the feed's earthquakes with a point inside the view.
func quakesIn(feed []globalfeed.Event, v tty.MapView) []globalfeed.Event {
	var out []globalfeed.Event
	for _, e := range feed {
		if e.Class == globalfeed.ClassQuake && e.HasPoint && v.Contains(e.Lat, e.Lon) {
			out = append(out, e)
		}
	}
	return out
}

// quakeOverlays is each quake as USGS draws it (D-123): a ring around its
// epicentre fixed on the screen by its magnitude, in its age's colour,
// labelled with its magnitude and its local time. No severity: the library
// never reports a quake as an alert over a place.
func quakeOverlays(quakes []globalfeed.Event, now time.Time, clock render.Clock) []tuimaps.Overlay {
	var out []tuimaps.Overlay
	for _, e := range quakes {
		mag := 0.0
		if e.Quake != nil && e.Quake.Mag != nil {
			mag = *e.Quake.Mag
		}
		out = append(out, tuimaps.Overlay{ID: quakeLayerKey + "/" + e.ID, Valid: e.At, Keeps: quakeKeeps,
			Features: []tuimaps.Feature{{Kind: tuimaps.Circle, Centre: tuimaps.LonLat{Lon: e.Lon, Lat: e.Lat},
				RadiusDots: quakeRingDots(mag), Role: quakeAgeRole(now.Sub(e.At)), Label: quakeLabel(mag, e.At, now, clock), ID: e.ID}}})
	}
	return out
}

// quakeRingDots is a ring's radius on the screen: half again with each
// magnitude, as USGS's circles grow - 3 dots at M2.5, 8 at M5, 17 at M7 -
// from 2 to 48.
func quakeRingDots(mag float64) int {
	return min(max(int(math.Round(math.Pow(1.5, mag))), 2), 48)
}

// quakeAgeRole is a quake's colour by its age, as USGS colours it: the past
// hour, the past day, older.
func quakeAgeRole(age time.Duration) tuimaps.Token {
	switch {
	case age < time.Hour:
		return tuimaps.QuakeHour
	case age < 24*time.Hour:
		return tuimaps.QuakeDay
	}
	return tuimaps.QuakeOlder
}

// quakeLabel is a quake's words: its magnitude and its local time in the
// listener's clock - "M4.1 2:14 PM" - and the day before it when not today.
func quakeLabel(mag float64, at, now time.Time, clock render.Clock) string {
	local := at.In(now.Location())
	when := clock.Time(local)
	if local.Format("2006-01-02") != now.Format("2006-01-02") {
		when = local.Format("Mon") + " " + when
	}
	return "M" + strconv.FormatFloat(mag, 'f', 1, 64) + " " + when
}
