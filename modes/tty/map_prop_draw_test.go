package tty

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/term"
)

// propSnapAt is a snapshot whose hour holds a field of each layer over the
// world's 2° grid: MUF(3000) mhz and foF2 a third of it, every cell with data.
func propSnapAt(at time.Time, mhz float32) ionomaps.Snapshot {
	field := func(v float32) ionomaps.Field {
		f := ionomaps.Field{West: -180, South: -90, East: 180, North: 90, Cols: 180, Rows: 90, Values: make([]float32, 180*90)}
		for i := range f.Values {
			f.Values[i] = v
		}
		return f
	}
	return ionomaps.Snapshot{Computed: at, Hours: []ionomaps.Hour{{At: at, MUF3000: field(mhz), FoF2: field(mhz / 3)}},
		Background: ionomaps.Backgrounds{FoF2: ionomaps.GloTEC, M3000: ionomaps.Climatology}, Inputs: ionomaps.Inputs{GloTECValid: at.Add(-30 * time.Minute)}}
}

// drawnDash is the map in the Propagation mode with a snapshot drawn.
func drawnDash(t *testing.T) (Dashboard, *seam) {
	t.Helper()
	s := &seam{answer: PropagationResult{Snapshot: propSnapAt(time.Now().UTC().Truncate(time.Minute), 18)}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	return runProp(t, d, cmd), s
}

// propGrid is the one Propagation grid handed to the library, and its preset.
func propGrid(t *testing.T, d Dashboard) *tuimaps.Grid {
	t.Helper()
	if len(d.mapPane.propGiven) != 1 {
		t.Fatalf("%d Propagation overlays handed in; want one, the hour on screen (D-112)", len(d.mapPane.propGiven))
	}
	for _, o := range d.mapPane.propGiven {
		return o.Grid
	}
	return nil
}

// TestBothPropagationLayersDrawWithTheirUnits is W3.1 (FR-1.5, D-24): the
// hour's MUF(3000) is drawn in MHz with its preset and legend; L draws foF2
// in its place; the legend row names the layer drawn.
func TestBothPropagationLayersDrawWithTheirUnits(t *testing.T) {
	d, _ := drawnDash(t)
	g := propGrid(t, d)
	if g.Type.Preset != "muf" || g.Type.Unit != "MHz" || g.Cols != 180 || g.Rows != 90 || g.Values[0] != 18 {
		t.Errorf("the grid is %+v at %v", g.Type, g.Values[0])
	}
	if row := stripANSITest(d.scrubRows(d.mapTextW())[0]); !strings.Contains(row, "MUF(3000), MHz") {
		t.Errorf("the legend row does not key MUF(3000) in MHz: %q", row)
	}
	d = pressMap(t, d, "L")
	if g = propGrid(t, d); g.Type.Preset != "fof2" || g.Type.Unit != "MHz" || g.Values[0] != 6 {
		t.Errorf("after L the grid is %+v at %v; want foF2", g.Type, g.Values[0])
	}
	if row := stripANSITest(d.scrubRows(d.mapTextW())[0]); !strings.Contains(row, "foF2, MHz") {
		t.Errorf("after L the legend row does not key foF2: %q", row)
	}
	if d = pressMap(t, d, "P"); len(d.mapPane.propGiven) != 0 {
		t.Errorf("leaving the mode left %d Propagation overlays drawn", len(d.mapPane.propGiven))
	}
}

// TestAPropagationFieldCrossesTheSea is W3.1 (FR-1.6, D-32): the field
// continues over the sea; a cell with no data is no value, never a number.
func TestAPropagationFieldCrossesTheSea(t *testing.T) {
	d, _ := drawnDash(t)
	if g := propGrid(t, d); !g.OverWater {
		t.Error("the Propagation field stops at the shore")
	}
	f := propSnapAt(time.Now(), 9).Hours[0].MUF3000
	f.NoData = ionomaps.Bitset{}
	if g := fieldGrid(f); len(g.Values) != 180*90 || math.IsNaN(g.Values[5]) {
		t.Fatalf("a whole field converted to %d values, the sixth %v", len(g.Values), g.Values[5])
	}
}

// TestEveryPropagationControlIsAnAction is W2.4 (FR-1.14, D-153, D-155,
// D-156): P, L and U are map actions Help lists and [keys] can rebind; in
// the weather modes L and U do nothing.
func TestEveryPropagationControlIsAnAction(t *testing.T) {
	km := defaultMapKeyMap()
	for act, key := range map[term.Action]string{actMapProp: "P", actMapLayer: "L", actMapRefresh: "U"} {
		if !slices.Equal(km[act].Keys, []string{key}) || !slices.Contains(mapActions, act) || km[act].Help == "" {
			t.Errorf("%s is %+v, listed %v; want %s", act, km[act], slices.Contains(mapActions, act), key)
		}
		if keys, err := mapKeysFrom(term.KeyMap{act: {Keys: []string{"ctrl+t"}}}); err != nil || !slices.Equal(keys[act].Keys, []string{"ctrl+t"}) {
			t.Errorf("[keys] did not rebind %s: %v %v", act, keys[act].Keys, err)
		}
	}
	var help []string
	for _, r := range mapHelpRows(km, false) {
		help = append(help, r.keys+"  "+r.help)
	}
	text := strings.Join(help, "\n")
	for _, want := range []string{"L", "U", "Layer", "Refresh"} {
		if !strings.Contains(text, want) {
			t.Errorf("Help does not list %q:\n%s", want, text)
		}
	}
	s := &seam{answer: PropagationResult{Err: errors.New("offline")}}
	d := propDash(t, s, true)
	before := d.mapPane.propLayer
	for _, k := range []string{"L", "U"} {
		var cmd tea.Cmd
		d, cmd = keyCmd(t, d, k)
		d = runProp(t, d, cmd)
	}
	if s.count() != 0 || d.mapPane.propLayer != before {
		t.Errorf("in a weather mode L and U asked %d times and set the layer to %d", s.count(), d.mapPane.propLayer)
	}
}

// TestHelpFitsWithThePropagationKeys is W2.4: Help with the new rows lists
// every map row and fits the terminal at the floor and at the 133x44 size.
func TestHelpFitsWithThePropagationKeys(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {133, 44}} {
		d := mapDash(t, Config{})
		m, _ := d.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		d = m.(Dashboard).open(modalHelp)
		text := strings.Join(strings.Fields(stripANSITest(strings.Join(d.modalLines(), " "))), " ") // wrapped at the floor: read as one text
		for _, want := range []string{"Radar Mode / Forecast Hi-Lo", "Propagation / Layer / Refresh Now", "Zoom In / Out / Scroll"} {
			if !strings.Contains(text, want) {
				t.Errorf("%dx%d: Help lacks %q", size[0], size[1], want)
			}
		}
		for _, l := range strings.Split(stripANSITest(d.View().Content), "\n") {
			if w := len([]rune(l)); w > size[0] {
				t.Errorf("%dx%d: a line of the frame with Help open is %d wide: %q", size[0], size[1], w, l)
			}
		}
	}
}

// TestEachControlsStateIsSaid is W2.4 (FR-1.14): the chips say the layer
// drawn and that the mode is on; the status says an update in flight.
func TestEachControlsStateIsSaid(t *testing.T) {
	d, _ := drawnDash(t)
	chips := stripANSITest(d.mapStatusLine())
	for _, want := range []string{"L] MUF(3000)", "U] Refresh Now", "P] Propagation On"} {
		if !strings.Contains(chips, want) {
			t.Errorf("the chips %q do not say %q", chips, want)
		}
	}
	if d = pressMap(t, d, "L"); !strings.Contains(stripANSITest(d.mapStatusLine()), "L] foF2") {
		t.Errorf("after L the chips say %q", stripANSITest(d.mapStatusLine()))
	}
	busy := &seam{wait: true}
	d, _ = keyCmd(t, propDash(t, busy, true), "P")
	if !strings.Contains(stripANSITest(d.mapStatusLine()), "asking") {
		t.Errorf("an update in flight is not said: %q", stripANSITest(d.mapStatusLine()))
	}
	_ = d.stopPropagation() // its context ended: the waiting seam returns
}

// TestURefreshesNowUnderTheThrottle is W2.4 (D-120, D-156): U asks for an
// update now; while one is in flight it asks nothing more; the library's
// early answer is said with its reason.
func TestURefreshesNowUnderTheThrottle(t *testing.T) {
	d, s := drawnDash(t)
	early := propSnapAt(time.Now().UTC().Truncate(time.Minute), 18)
	early.Early = ionomaps.TooSoon
	s.mu.Lock()
	s.answer = PropagationResult{Snapshot: early}
	s.mu.Unlock()
	d, cmd := keyCmd(t, d, "U")
	if d = runProp(t, d, cmd); s.count() != 2 {
		t.Fatalf("U asked %d in all; want a second ask", s.count())
	}
	if !strings.Contains(stripANSITest(d.mapStatusLine()), "too soon") {
		t.Errorf("the early answer's reason is not said: %q", stripANSITest(d.mapStatusLine()))
	}
	held := &seam{wait: true}
	d, _ = keyCmd(t, propDash(t, held, true), "P")
	d, cmd = keyCmd(t, d, "U")
	if cmd != nil && len(propMsgs(cmd)) > 0 || d.mapPane.propBusy != true {
		t.Error("U with an update in flight asked again")
	}
	_ = d.stopPropagation()
}
