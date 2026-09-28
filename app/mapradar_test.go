package app

// mapradar_test.go — 0.18.0 W8 at the app: the loop's slots (FR-5.2), the
// source by region (D-83), the newest frame first (W8.12), a failed frame a
// gap, the estimate (W8.15) and the hosts named (FR-3.8, D-75).

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// fakeRadar is a source with its times and a frame for every one, failing
// the times listed in fail; asks counts the frames fetched, peak the most in
// flight at once. A frame takes delay, and first waits for gate if set.
type fakeRadar struct {
	name    string
	regions []string
	times   []time.Time
	fail    map[time.Time]bool
	png     []byte
	delay   time.Duration
	gate    chan struct{}
	mu      sync.Mutex
	asks    int
	flying  int
	peak    int
}

func (f *fakeRadar) Name() string { return f.name }
func (f *fakeRadar) Covers(r string) bool {
	for _, c := range f.regions {
		if c == r {
			return true
		}
	}
	return false
}
func (f *fakeRadar) Times(context.Context, string) ([]time.Time, error) { return f.times, nil }
func (f *fakeRadar) Frame(_ context.Context, _ string, at time.Time, _ []time.Time, _ radar.Box) ([]byte, error) {
	f.mu.Lock()
	f.asks++
	f.flying++
	f.peak = max(f.peak, f.flying)
	f.mu.Unlock()
	defer func() { f.mu.Lock(); f.flying--; f.mu.Unlock() }()
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-time.After(time.Second):
			return nil, errors.New("waited a second for the hours ahead to be asked")
		}
	}
	time.Sleep(f.delay)
	if f.fail[at] {
		return nil, errors.New("refused")
	}
	return f.png, nil
}

func grid5(n int, from time.Time) []time.Time {
	var out []time.Time
	for i := range n {
		out = append(out, from.Add(time.Duration(i)*5*time.Minute))
	}
	return out
}

func TestTheLoopsSlotsAreTheSourcesTimes(t *testing.T) {
	start := time.Date(2026, 9, 26, 14, 0, 0, 0, time.UTC)
	slots := loopSlots(grid5(30, start), radarStep, radar.Window)
	if len(slots) != 24 {
		t.Fatalf("%d slots over two hours at five minutes", len(slots))
	}
	for i, s := range slots {
		if s.time.IsZero() || !s.time.Equal(s.at) {
			t.Errorf("IEM's grid slot %d is %v filled by %v", i, s.at, s.time)
		}
	}
	// MRMS: about every two minutes, and a twenty-minute hole
	var mrms []time.Time
	for m := 0; m < 125; m += 2 {
		if m >= 60 && m < 80 {
			continue
		}
		mrms = append(mrms, start.Add(time.Duration(m)*time.Minute+17*time.Second))
	}
	gaps := 0
	for _, s := range loopSlots(mrms, radarStep, radar.Window) {
		if s.time.IsZero() {
			gaps++
		} else if s.at.Sub(s.time) >= radarStep || s.time.After(s.at) {
			t.Errorf("slot %v filled by %v, not the newest within a step before it", s.at, s.time)
		}
	}
	if gaps < 3 || gaps > 5 {
		t.Errorf("a twenty-minute hole made %d gaps; want about four, stated", gaps)
	}
}

func TestTheSourceIsMRMSUnlessIEMIsChosenForTheLower48(t *testing.T) {
	png, err := os.ReadFile("testdata/radar-frame.png")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 26, 16, 0, 0, 0, time.UTC)
	times := grid5(30, now.Add(-150*time.Minute))
	iem := &fakeRadar{name: "IEM", regions: []string{geo.RegionContiguous}, times: times, png: png}
	mrms := &fakeRadar{name: "MRMS", regions: []string{geo.RegionContiguous, geo.RegionHawaii}, times: times, png: png,
		fail: map[time.Time]bool{times[len(times)-2]: true}}
	lp := &livePipelines{radar: &radarSources{iem: iem, mrms: mrms}}
	socal := tty.MapView{W: -119.6, S: 32.1, E: -115.0, N: 34.2}
	got := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionContiguous, View: socal})
	if got.Source != "MRMS" || got.Note != "" || len(got.Overlays) == 0 { // its colours approximate: the Status window's and the chip's MRMS≈ (D-132)
		t.Fatalf("the default is %+v", got)
	}
	for _, o := range got.Overlays {
		if !strings.HasPrefix(o.ID, tty.RadarLayer+"/us-") || o.Image == nil || len(o.Image.Frames) != 24 {
			t.Fatalf("the loop is %s with %v", o.ID, o.Image)
		}
		if f := o.Image.Frames[22]; !f.Gap || f.PNG != nil {
			t.Errorf("the failed frame is %+v, not a gap", f)
		}
		if f := o.Image.Frames[23]; f.Gap || len(f.PNG) == 0 {
			t.Error("the newest frame is missing")
		}
	}
	if got := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionContiguous, View: socal, RadarIEM: true}); got.Source != "IEM" || got.Note != "" {
		t.Errorf("IEM chosen for the lower 48 gives %s (%q)", got.Source, got.Note)
	}
	if got := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionHawaii, View: socal, RadarIEM: true}); got.Source != "MRMS" {
		t.Errorf("Hawaii with IEM chosen gives %q; IEM does not cover it", got.Source)
	}
	if got := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionSamoa, View: socal}); got.Source != "" || got.Note != "No radar covers American Samoa." || len(got.Overlays) != 0 {
		t.Errorf("American Samoa gives %+v", got)
	}
	was := iem.asks
	whole := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionContiguous, View: socal, RadarIEM: true})
	if iem.asks-was != 24*len(whole.Overlays) || len(whole.Overlays) == 0 || len(whole.Overlays[0].Image.Frames) != 24 {
		t.Errorf("the whole loop fetched %d frames for %d boxes; want 24 a box (D-85)", iem.asks-was, len(whole.Overlays))
	}
}

func TestTheRadarsCostAndHostsAreSaid(t *testing.T) {
	b, r := radarLayerCost(mapInputs{region: geo.RegionContiguous, view: tty.MapView{W: -125, S: 24, E: -66, N: 50}})
	if r != 25 || b != 24*radarFrameBytes {
		t.Errorf("the whole lower 48 costs %d bytes in %d requests; want one box of 24 frames and the time list", b, r)
	}
	if b, r := radarLayerCost(mapInputs{region: geo.RegionSamoa}); b != 0 || r != 0 {
		t.Error("a region no radar covers costs something")
	}
	hosts := map[string]bool{}
	for _, s := range mapSourceList() {
		hosts[s.Host] = true
	}
	for _, h := range []string{"mesonet.agron.iastate.edu", "opengeo.ncep.noaa.gov"} {
		if !hosts[h] {
			t.Errorf("the Status window does not name %s", h)
		}
	}
}

// TestALoopShowingNothingIsCheckedAgainstTheOtherSource is D-84's check: a
// loop whose every frame is empty while the other source shows echo in the
// same box is flagged; an empty loop both agree on is a clear sky.
func TestALoopShowingNothingIsCheckedAgainstTheOtherSource(t *testing.T) {
	echo, err := os.ReadFile("testdata/radar-frame.png")
	if err != nil {
		t.Fatal(err)
	}
	clear, err := os.ReadFile("testdata/radar-empty.png")
	if err != nil {
		t.Fatal(err)
	}
	times := grid5(30, time.Date(2026, 9, 26, 13, 30, 0, 0, time.UTC))
	socal := tty.MapAsk{Region: geo.RegionContiguous, View: tty.MapView{W: -119.6, S: 32.1, E: -115.0, N: 34.2}, RadarIEM: true}
	iem := &fakeRadar{name: "IEM", regions: []string{geo.RegionContiguous}, times: times, png: clear}
	mrms := &fakeRadar{name: "MRMS", regions: []string{geo.RegionContiguous}, times: times, png: echo}
	lp := &livePipelines{radar: &radarSources{iem: iem, mrms: mrms}}
	if got := lp.mapRadar(context.Background(), socal); !strings.Contains(got.Note, "IEM shows no echo where MRMS does") {
		t.Errorf("an empty loop beside the other source's echo says %q", got.Note)
	}
	mrms.png = clear
	if got := lp.mapRadar(context.Background(), socal); got.Note != "" || len(got.Overlays) == 0 {
		t.Errorf("a clear sky both agree on says %q with %d loops", got.Note, len(got.Overlays))
	}
}

// TestTheLoopFitsTheBudget is D-88 and W8.6: a loop over the budget as
// fetched is trimmed of its oldest frames, never refused whole.
func TestTheLoopFitsTheBudget(t *testing.T) {
	png, err := os.ReadFile("testdata/radar-frame.png")
	if err != nil {
		t.Fatal(err)
	}
	per := int64(len(png)) + pixelsOf(png)
	count := int(radarBudgetShare/per) + 10 // ten frames past what the budget holds
	var frames []tuimaps.LoopFrame
	for i := range count {
		frames = append(frames, tuimaps.LoopFrame{Valid: time.Unix(int64(i)*300, 0), PNG: png})
	}
	loops := trimToBudget([]tuimaps.Overlay{{ID: "radar/x", Image: &tuimaps.Image{Frames: frames}}})
	kept := loops[0].Image.Frames
	if int64(len(kept))*per > radarBudgetShare || len(kept) < count-11 {
		t.Errorf("%d of %d frames kept at %d bytes each against %d", len(kept), count, per, radarBudgetShare)
	}
	if !kept[len(kept)-1].Valid.Equal(frames[len(frames)-1].Valid) {
		t.Error("the trim dropped the newest frame, not the oldest")
	}
}

// TestTheRadarAsksForAndKeepsWhatFits is D-88's wiring: a view across four
// boxes whose frames outgrow the budget is trimmed as fetched, so what is
// handed in fits it.
func TestTheRadarAsksForAndKeepsWhatFits(t *testing.T) {
	var big bytes.Buffer
	if err := png.Encode(&big, image.NewAlpha(image.Rect(0, 0, 499, 499))); err != nil {
		t.Fatal(err)
	}
	times := grid5(30, time.Date(2026, 9, 26, 13, 30, 0, 0, time.UTC))
	src := &fakeRadar{name: "MRMS", regions: []string{geo.RegionContiguous}, times: times, png: big.Bytes()}
	lp := &livePipelines{radar: &radarSources{iem: &fakeRadar{name: "IEM"}, mrms: src}}
	corners := tty.MapAsk{Region: geo.RegionContiguous, View: tty.MapView{W: -97, S: 35, E: -94, N: 39}}
	boxes := radar.BoxesFor(corners.Region, corners.View)
	got := lp.mapRadar(context.Background(), corners)
	if len(boxes) != 4 || len(got.Overlays) != 4 {
		t.Fatalf("a view across four boxes gave %d loops of %d boxes", len(got.Overlays), len(boxes))
	}
	total := int64(0)
	for _, o := range got.Overlays {
		for _, f := range o.Image.Frames {
			if !f.Gap {
				total += int64(len(f.PNG)) + pixelsOf(f.PNG)
			}
		}
	}
	if total > radarBudgetShare || len(got.Overlays[0].Image.Frames) >= 24 {
		t.Errorf("handed in %d bytes against %d, %d frames a loop", total, radarBudgetShare, len(got.Overlays[0].Image.Frames))
	}
}

// hrrrGet answers HRRR's run and its frames from the fixtures, and counts
// the frames.
type hrrrGet struct {
	t      *testing.T
	mu     sync.Mutex
	frames int
	run    []byte        // the run's answer, when not the fixture's
	asked  chan struct{} // closed at the first frame asked, when set
	once   sync.Once
}

func (h *hrrrGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	if strings.HasSuffix(rawURL, ".json") {
		if h.run != nil {
			return h.run, nil
		}
		return os.ReadFile("../domains/radar/testdata/hrrr-run.json")
	}
	if h.asked != nil {
		h.once.Do(func() { close(h.asked) })
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.frames++
	return os.ReadFile("../domains/radar/testdata/hrrr-frame.png")
}

// TestTheLoopRunsOnPastNow is D-113 and D-114: after the observed loop, a
// forecast loop a box of HRRR's quarter-hours after the newest observed frame
// and up to the horizon, every frame marked forecast; the observed loops drawn
// until their newest frame and the forecast from its first, so no moment
// shows both; HRRR named.
func TestTheLoopRunsOnPastNow(t *testing.T) {
	get := &hrrrGet{t: t}
	h := radar.NewHRRR(get, "")
	box := radar.Box{Name: "us", W: -126, S: 23, E: -65, N: 51, Cols: 600, Rows: 276}
	newest := time.Date(2026, 9, 27, 19, 40, 0, 0, time.UTC)
	observed := tuimaps.RadarImage(tty.RadarLayer+"/us", tuimaps.Image{Frames: []tuimaps.LoopFrame{{Valid: newest, Gap: true}}, Provider: tuimaps.ProviderIEM,
		West: box.W, South: box.S, East: box.E, North: box.N, Projection: tuimaps.PlateCarree}, newest)
	out := withForecast(context.Background(), tty.MapRadar{Overlays: []tuimaps.Overlay{observed}, Source: "MRMS"}, h, []radar.Box{box}, newest, newest.Add(3*time.Hour))
	if out.Ahead != "HRRR" || len(out.Overlays) != 2 {
		t.Fatalf("ahead %q, %d loops; want HRRR and a forecast loop beside the observed", out.Ahead, len(out.Overlays))
	}
	if out.Overlays[0].During.Until != newest {
		t.Errorf("the observed loop is drawn during %v; want until its newest frame", out.Overlays[0].During)
	}
	fc := out.Overlays[1]
	if !strings.HasPrefix(fc.ID, tty.RadarLayer+"/fc-") || fc.During.From != fc.Image.Frames[0].Valid {
		t.Errorf("the forecast loop %s is drawn during %v", fc.ID, fc.During)
	}
	if n := len(fc.Image.Frames); n != 12 || get.frames != 12 {
		t.Errorf("%d forecast frames (%d asked); want three hours of quarter-hours, 12", n, get.frames)
	}
	for _, f := range fc.Image.Frames {
		if !f.Forecast || !f.Valid.After(newest) || f.Valid.After(newest.Add(3*time.Hour)) {
			t.Fatalf("a forecast frame at %v (forecast %v) is outside the hours ahead", f.Valid, f.Forecast)
		}
	}
}

// TestTheHoursAheadFitWhatTheLoopLeaves is D-114: the forecast frames fit the
// budget the observed loops leave, the farthest dropped first.
func TestTheHoursAheadFitWhatTheLoopLeaves(t *testing.T) {
	png, _ := os.ReadFile("../domains/radar/testdata/hrrr-frame.png")
	frames := func() []tuimaps.LoopFrame {
		var out []tuimaps.LoopFrame
		for i := range 8 {
			out = append(out, tuimaps.LoopFrame{Valid: time.Date(2026, 9, 27, 20, 15*i, 0, 0, time.UTC), PNG: png, Forecast: true})
		}
		return out
	}
	loops := []tuimaps.Overlay{{ID: "radar/fc-a", Image: &tuimaps.Image{Frames: frames()}}}
	one := chargeOf(loops) / 8
	kept := trimForecast(loops, one*5)
	if n := len(kept[0].Image.Frames); n != 5 {
		t.Fatalf("%d frames kept in room for 5", n)
	}
	if !kept[0].Image.Frames[4].Valid.Equal(time.Date(2026, 9, 27, 21, 0, 0, 0, time.UTC)) {
		t.Errorf("the frames kept end at %v; want the nearest five, the farthest dropped", kept[0].Image.Frames[4].Valid)
	}
}

// TestTheLoopsFramesAreFetchedSixAtATime is D-130 (UAT-2 U2-35): the frames
// were fetched one after another, a frame every 200 ms at the client's pace,
// the lower 48's loop 4.8 s cold. Six at a time, as the zones are (D-46), each
// in its place in the loop.
func TestTheLoopsFramesAreFetchedSixAtATime(t *testing.T) {
	png, _ := os.ReadFile("../domains/radar/testdata/hrrr-frame.png")
	times := grid5(24, time.Date(2026, 9, 28, 14, 0, 0, 0, time.UTC))
	src := &fakeRadar{name: "MRMS", times: times, png: png, delay: 20 * time.Millisecond}
	o, _, ok := radarLoop(context.Background(), src, geo.RegionContiguous, times, loopSlots(times, radarStep, radar.Window), radar.Box{Name: "us", W: -126, S: 23, E: -65, N: 51})
	if !ok {
		t.Fatal("no loop")
	}
	if src.peak != 6 || src.asks != len(o.Image.Frames) { // six, as D-130 ruled
		t.Errorf("%d frames fetched, at most %d at once; want every frame, six at once", src.asks, src.peak)
	}
	for i, f := range o.Image.Frames {
		if f.Gap || (i > 0 && !f.Valid.After(o.Image.Frames[i-1].Valid)) {
			t.Fatalf("frame %d is %+v: out of its place, or missing", i, f.Valid)
		}
	}
}

// TestTheHoursAheadAreFetchedAlongsideTheLoop is D-130: HRRR's hours ahead
// were asked only once the observed loop was whole - 2.7 s after its 4.8.
// They are asked alongside: here the observed frames wait for HRRR's first
// ask, and a loop fetched first would give up waiting.
func TestTheHoursAheadAreFetchedAlongsideTheLoop(t *testing.T) {
	png, _ := os.ReadFile("../domains/radar/testdata/hrrr-frame.png")
	now := time.Now().UTC()
	times := grid5(24, now.Truncate(5*time.Minute).Add(-115*time.Minute))
	get := &hrrrGet{t: t, asked: make(chan struct{}),
		run: []byte(`{"model_init_utc": "` + now.Truncate(time.Hour).Add(-time.Hour).Format(time.RFC3339) + `"}`)}
	src := &fakeRadar{name: "MRMS", regions: []string{geo.RegionContiguous}, times: times, png: png, gate: get.asked}
	lp := &livePipelines{radar: &radarSources{iem: &fakeRadar{name: "IEM"}, mrms: src, hrrr: radar.NewHRRR(get, "")}}
	out := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionContiguous, View: tty.MapView{W: -125, S: 24, E: -66, N: 50}, RadarAhead: 3})
	if out.Ahead != "HRRR" || len(out.Overlays) < 2 {
		t.Errorf("ahead %q, %d loops (problems %v); want the observed loop and HRRR's beside it", out.Ahead, len(out.Overlays), out.Problems)
	}
}
