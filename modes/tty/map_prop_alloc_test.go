package tty

// map_prop_alloc_test.go — the Propagation mode's frame pinned as the
// weather modes' is (FR-8.3): a memo hit, a miss, colour off and on, and a
// draw.

import (
	"math"
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// propAllocBudget pins the Propagation window over the 133×44 fixture, its
// block under the map drawn, over the shaped field (D-282): measured and set
// at × 1.05, as the other pins are. D-194's window, inset to inset and 85%
// wide, draws more map rows than U1-13's, and every row costs its
// allocations.
var propAllocBudget = map[string]float64{"hit": 1_280 * 1.05, "miss": 1_742 * 1.05, "hit colour": 1_762 * 1.05, "miss colour": 3_415 * 1.05}

// propShaped is propSnapAt's snapshot with its MUF(3000) shaped as a
// recorded map is, go-tuiMaps' benchmark field (D-261, D-282): highest on the
// day side, its crests either side of the equator, structure along the
// longitudes and two closed patches, 4 to 30 MHz; foF2 a third of it.
func propShaped(at time.Time) ionomaps.Snapshot {
	s := propSnapAt(at, 18)
	r := math.Pi / 180
	for i := range s.Now.MUF3000.Values {
		lat, lon := 89-2*float64(i/180), -179+2*float64(i%180)
		day := 1 + 0.6*math.Cos((lon-30)*r)
		crests := 1 + 0.25*math.Cos(4*lat*r)
		v := 6 + 14*math.Cos(lat*r)*day*crests + 2*math.Sin(4*lon*r)*math.Cos(3*lat*r)
		v += 5*math.Exp(-((lat-35)*(lat-35)+(lon+100)*(lon+100))/150) - 4*math.Exp(-((lat+30)*(lat+30)+(lon-140)*(lon-140))/200)
		s.Now.MUF3000.Values[i] = float32(min(max(v, 4), 30))
		s.Now.FoF2.Values[i] = s.Now.MUF3000.Values[i] / 3
	}
	return s
}

// propBench is the map in the Propagation mode, a shaped snapshot drawn, its
// clock pinned so the age holds still, every piece of work landed.
func propBench(tb testing.TB) Dashboard {
	tb.Helper()
	t := tb
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	s := &seam{answer: PropagationResult{Snapshot: propShaped(at)}}
	var asked []string
	d := openMap(t, Config{PropagationUpdate: s.update, PropagationAck: propAckVersion, MapRadar: radarFeed(t, "MRMS", &asked)}, 133, 44)
	d.now = func() time.Time { return at.Add(7 * time.Minute) }
	d, cmd := keyCmd(t, d, "P")
	d = runProp(t, d, cmd)
	return settleWork(d)
}

// settleWork lands every piece of the map's work.
func settleWork(d Dashboard) Dashboard {
	for cmd := d.mapWorkCmd(); cmd != nil; cmd = d.mapWorkCmd() {
		m, _ := d.Update(cmd())
		d = m.(Dashboard)
	}
	return d
}

func BenchmarkFrame_133x44_MapPropagation(b *testing.B) {
	d := propBench(b)
	b.ReportAllocs()
	for b.Loop() {
		d.mmemo.ok = false
		_ = d.View().Content
	}
}

// BenchmarkPropRenderMap is the frame the host draws as the map moves
// (D-282): each op pans one step, west and east in turn, lands the work the
// pan asks for, and draws the map over the shaped field - never an unchanged
// view, which measures the row cache rather than a frame.
func BenchmarkPropRenderMap(b *testing.B) {
	d := propBench(b)
	b.ReportAllocs()
	west := true
	for b.Loop() {
		key := map[bool]string{true: "left", false: "right"}[west]
		west = !west
		d, _ = keyCmd(b, d, key)
		d = settleWork(d)
		_ = d.renderMap()
	}
}

func TestPropagationFrameAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := propBench(t)
	if !strings.Contains(stripANSITest(d.View().Content), "MODE:") {
		t.Fatal("the fixture drew no Propagation block: this measures nothing")
	}
	for _, colour := range []bool{false, true} {
		rendering.SetColorEnabledForTest(colour)
		name := map[bool]string{false: "", true: " colour"}[colour]
		calls := &[]string{}
		d.mapPane.calls = calls
		hit := testing.AllocsPerRun(20, func() { _ = d.View().Content })
		miss := testing.AllocsPerRun(20, func() { d.mmemo.ok = false; _ = d.View().Content })
		if len(*calls) != 0 {
			t.Errorf("a frame called the library: %v", *calls)
		}
		t.Logf("Propagation window%s: hit %.0f allocs (budget %.0f) · miss %.0f allocs (budget %.0f)", name, hit, propAllocBudget["hit"+name], miss, propAllocBudget["miss"+name])
		if hit > propAllocBudget["hit"+name] {
			t.Errorf("the Propagation window%s (memo hit) allocates %.0f per View(), budget %.0f", name, hit, propAllocBudget["hit"+name])
		}
		if miss > propAllocBudget["miss"+name] {
			t.Errorf("the Propagation window%s (memo miss) allocates %.0f per View(), budget %.0f", name, miss, propAllocBudget["miss"+name])
		}
	}
	rendering.SetColorEnabledForTest(false)
}
