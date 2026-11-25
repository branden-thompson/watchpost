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
	At         time.Time
	LoopIndex  int
	LoopCount  int
	LoopAt     time.Time
	Playing    bool
	Lon, Lat   float64
	Zoom       float64
	RadarCells int // cells drawn in the radar's colours
	// Cells are the cells drawn in each legend preset's colours - the
	// temperature's, the wind's - so a layer missing from a frame of the
	// loop shows as a zero (U2-55 to U2-58).
	Cells map[string]int
	// Given are the temperature feed's overlays as handed in, by layer -
	// what was delivered, beside what was drawn (U2-55 to U2-58).
	Given map[string]int
	// Text is the frame's map as plain text, kept while
	// WATCHPOST_DEBUG_MAPFRAMES_TEXT is set: a field's contour values and
	// the wind's arrows read from it.
	Text string `json:",omitempty"`
	// Spans are the temperature feed's overlays handed in, each id with its
	// span, and Chips and Notes the feed's - kept with Text.
	Spans        map[string]string   `json:",omitempty"`
	Chips        map[string][]string `json:",omitempty"`
	Notes        []string            `json:",omitempty"`
	Places       []string            // the place names on the frame, sorted
	Status       string              // the library's: complete, sharpening, ...
	PendingAfter bool                // work still pending once drawn
	// Radar is the radar's loops as handed in, by id, sorted: which boxes
	// the frame was drawn from (U2-59). RadarFrames is each loop's frames
	// that carry a picture, and its gaps: "frames/gaps".
	Radar       []string
	RadarFrames map[string]string `json:",omitempty"`
	// Problem, on a record of its own, is a problem said to the
	// diagnostics: in the file between the frames it fell between.
	Problem string `json:",omitempty"`
}

// placeWords are the words a frame's place names are made of.
var placeWords = regexp.MustCompile(`[A-Z][A-Za-z'.-]{2,}(?: [A-Z][A-Za-z'.-]+)*`)

// recordFrame tells the recorder the frame just drawn; nothing without one.
func (mm mapModel) recordFrame(status tuimaps.Status, e mapEnv) {
	if mm.mapHooks.MapFrame == nil || mm.mapPane.m == nil {
		return
	}
	st := mm.mapPane.m.Loop()
	c, z := mm.mapPane.m.Centre()
	plain := plaintext.StripSGR(strings.Join(mm.mapPane.lines, "\n"))
	seen := map[string]bool{}
	for _, w := range placeWords.FindAllString(plain, -1) {
		seen[w] = true
	}
	places := make([]string, 0, len(seen))
	for w := range seen {
		places = append(places, w)
	}
	sort.Strings(places)
	mm.mapHooks.MapFrame(MapFrame{At: e.Now(), LoopIndex: st.Index, LoopCount: st.Count, LoopAt: st.At, Playing: st.Playing,
		Lon: c.Lon, Lat: c.Lat, Zoom: z, RadarCells: mm.radarCells(), Cells: mm.presetCells(), Given: mm.tempGivenByLayer(), Text: mm.frameText(plain), Spans: mm.givenSpans(), Chips: mm.frameChips(), Notes: mm.frameNotes(),
		Places: places, Status: strconv.Itoa(int(status)), PendingAfter: mm.mapPane.pending, Radar: mm.radarGivenIDs(), RadarFrames: mm.radarFrameCounts()})
}

// radarGivenIDs is the radar's loops handed in, by id, sorted.
func (mm mapModel) radarGivenIDs() []string {
	out := make([]string, 0, len(mm.mapPane.radarGiven))
	for id := range mm.mapPane.radarGiven {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// radarFrameCounts is each radar loop's frames with a picture and its gaps,
// by id.
func (mm mapModel) radarFrameCounts() map[string]string {
	out := map[string]string{}
	for id, o := range mm.mapPane.radarGiven {
		if o.Image == nil {
			continue
		}
		gaps := 0
		for _, f := range o.Image.Frames {
			if f.Gap {
				gaps++
			}
		}
		out[id] = strconv.Itoa(len(o.Image.Frames)-gaps) + "/" + strconv.Itoa(gaps)
	}
	return out
}

// tempGivenByLayer counts the temperature feed's overlays by layer.
func (mm mapModel) tempGivenByLayer() map[string]int {
	t := mm.mapPane.temp
	return map[string]int{TemperatureLayer: len(t.Overlays) + len(t.High) + len(t.Low), FeelsLayer: len(t.Feels) + len(t.FeelsHigh) + len(t.FeelsLow),
		WindLayer: len(t.Wind) + len(t.WindDays), UVLayer: len(t.UV) + len(t.UVDays), AirLayer: len(t.Air) + len(t.AirDays)}
}

// givenSpans are the temperature overlays handed in and their spans, while
// the recorder keeps text.
func (mm mapModel) givenSpans() map[string]string {
	if !mm.mapHooks.MapFrameText {
		return nil
	}
	out := map[string]string{}
	for id, o := range mm.mapPane.tempGiven {
		out[id] = o.During.From.UTC().Format("15:04") + ".." + o.During.Until.UTC().Format("15:04")
	}
	return out
}

// frameChips and frameNotes are the temperature feed's, while the recorder
// keeps text.
func (mm mapModel) frameChips() map[string][]string {
	if !mm.mapHooks.MapFrameText {
		return nil
	}
	return mm.mapPane.temp.Chips
}

func (mm mapModel) frameNotes() []string {
	if !mm.mapHooks.MapFrameText {
		return nil
	}
	return append(append([]string(nil), mm.mapPane.temp.Notes...), mm.mapPane.temp.Problems...)
}

// frameText is the frame's plain text while the recorder is asked for it.
func (mm mapModel) frameText(plain string) string {
	if !mm.mapHooks.MapFrameText {
		return ""
	}
	return plain
}

// radarCells counts the frame's cells drawn in one of the radar's colours.
func (mm mapModel) radarCells() int { return mm.presetCells()["radar"] }

// presetCells counts the frame's cells drawn in each legend preset's colours.
func (mm mapModel) presetCells() map[string]int {
	text := strings.Join(mm.mapPane.lines, "\n")
	out := map[string]int{}
	for _, e := range mm.mapPane.legend {
		for _, c := range e.Classes {
			if c.Drawn {
				out[e.Preset] += strings.Count(text, strconv.Itoa(int(c.Colour.R))+";"+strconv.Itoa(int(c.Colour.G))+";"+strconv.Itoa(int(c.Colour.B))+"m")
			}
		}
	}
	return out
}
