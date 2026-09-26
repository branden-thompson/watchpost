package app

// mapradar_test.go — 0.18.0 W8 at the app: the loop's slots (FR-5.2), the
// source by region (D-83), the newest frame first (W8.12), a failed frame a
// gap, the estimate (W8.15) and the hosts named (FR-3.8, D-75).

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
)

// fakeRadar is a source with its times and a frame for every one, failing
// the times listed in fail; asks counts the frames fetched.
type fakeRadar struct {
	name    string
	regions []string
	times   []time.Time
	fail    map[time.Time]bool
	png     []byte
	asks    int
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
	f.asks++
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
	got := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionContiguous, View: socal}, false)
	if got.Source != "MRMS" || !strings.Contains(got.Note, "approximate") || len(got.Overlays) == 0 {
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
	if got := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionContiguous, View: socal, RadarIEM: true}, false); got.Source != "IEM" || got.Note != "" {
		t.Errorf("IEM chosen for the lower 48 gives %s (%q)", got.Source, got.Note)
	}
	if got := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionHawaii, View: socal, RadarIEM: true}, false); got.Source != "MRMS" {
		t.Errorf("Hawaii with IEM chosen gives %q; IEM does not cover it", got.Source)
	}
	if got := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionSamoa, View: socal}, false); got.Source != "" || got.Note != "No radar covers American Samoa." || len(got.Overlays) != 0 {
		t.Errorf("American Samoa gives %+v", got)
	}
	was := iem.asks
	newest := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionContiguous, View: socal, RadarIEM: true}, true)
	if iem.asks-was != len(newest.Overlays) || len(newest.Overlays) == 0 || len(newest.Overlays[0].Image.Frames) != 1 {
		t.Errorf("the newest-only ask fetched %d frames for %d boxes", iem.asks-was, len(newest.Overlays))
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
	if got := lp.mapRadar(context.Background(), socal, false); !strings.Contains(got.Note, "IEM shows no echo where MRMS does") {
		t.Errorf("an empty loop beside the other source's echo says %q", got.Note)
	}
	mrms.png = clear
	if got := lp.mapRadar(context.Background(), socal, false); got.Note != "" || len(got.Overlays) == 0 {
		t.Errorf("a clear sky both agree on says %q with %d loops", got.Note, len(got.Overlays))
	}
}
