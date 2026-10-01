package tty

// map_framelog.go — the map's frame recorder (D-198): under
// WATCHPOST_DEBUG_MAPFRAMES the app hands a hook, and every map frame drawn
// is told to it - the loop's place, the view, how many cells are in the
// radar's colours, and the place names drawn - so U2-46's labels flipping and
// U2-47's radar missing at rest can be read from a live run. Without the
// switch there is no hook, and a frame pays one nil check.

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/plaintext"
)

// MapFrame is one drawn map frame, as the recorder keeps it.
type MapFrame struct {
	At           time.Time
	LoopIndex    int
	LoopCount    int
	LoopAt       time.Time
	Playing      bool
	Lon, Lat     float64
	Zoom         float64
	RadarCells   int      // cells drawn in the radar's colours
	Places       []string // the place names on the frame, sorted
	Status       string   // the library's: complete, sharpening, ...
	PendingAfter bool     // work still pending once drawn
}

// placeWords are the words a frame's place names are made of.
var placeWords = regexp.MustCompile(`[A-Z][A-Za-z'.-]{2,}(?: [A-Z][A-Za-z'.-]+)*`)

// recordFrame tells the recorder the frame just drawn; nothing without one.
func (d Dashboard) recordFrame(status tuimaps.Status) {
	if d.cfg.MapFrame == nil || d.mapPane.m == nil {
		return
	}
	st := d.mapPane.m.Loop()
	c, z := d.mapPane.m.Centre()
	plain := plaintext.StripSGR(strings.Join(d.mapPane.lines, "\n"))
	seen := map[string]bool{}
	for _, w := range placeWords.FindAllString(plain, -1) {
		seen[w] = true
	}
	places := make([]string, 0, len(seen))
	for w := range seen {
		places = append(places, w)
	}
	sort.Strings(places)
	d.cfg.MapFrame(MapFrame{At: d.now(), LoopIndex: st.Index, LoopCount: st.Count, LoopAt: st.At, Playing: st.Playing,
		Lon: c.Lon, Lat: c.Lat, Zoom: z, RadarCells: d.radarCells(), Places: places, Status: strconv.Itoa(int(status)), PendingAfter: d.mapPane.pending})
}

// radarCells counts the frame's cells drawn in one of the radar's colours.
func (d Dashboard) radarCells() int {
	text := strings.Join(d.mapPane.lines, "\n")
	n := 0
	for _, e := range d.mapPane.legend {
		if e.Preset != "radar" {
			continue
		}
		for _, c := range e.Classes {
			if c.Drawn {
				n += strings.Count(text, strconv.Itoa(int(c.Colour.R))+";"+strconv.Itoa(int(c.Colour.G))+";"+strconv.Itoa(int(c.Colour.B))+"m")
			}
		}
	}
	return n
}
