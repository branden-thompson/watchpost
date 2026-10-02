package tty

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// map_radar_swap_test.go — UAT-2 U2-59: a zoom that changes the radar's boxes
// keeps the radar on the map. The lower 48 is one box when the view is wide
// and up to four closer boxes when it is narrow, so a zoom across that line
// hands in loops under new ids and takes the old ones off.

// overBudget is the "N over its image budget" a refused hand-in says.
var overBudget = regexp.MustCompile(`([\d,]+) over its image budget`)

// renamed is the answer's loops under the ids of other boxes, and frames
// times as many frames: what a zoom across the wide line brings.
func renamed(r MapRadar, box string, frames int) MapRadar {
	out := r
	out.Overlays = nil
	for _, o := range r.Overlays {
		img := *o.Image
		for range frames - 1 {
			img.Frames = append(img.Frames, o.Image.Frames...)
		}
		o.Image, o.ID = &img, RadarLayer+"/"+box
		out.Overlays = append(out.Overlays, o)
	}
	return out
}

// budgetForOne sets the map's image budget to what it holds now plus the
// next loop, less one byte: the held loop fits, and the held loop and the
// next one together do not. The library says by how much a hand-in goes
// over, so the figure is read from a refusal made at a budget of one.
func budgetForOne(t *testing.T, d Dashboard, next tuimaps.Overlay) {
	t.Helper()
	var err error
	d.mapPane.call("SetImageBudget", func() { _ = d.mapPane.m.SetImageBudget(1) })
	d.mapPane.call("Set", func() { _, err = d.mapPane.m.Set(next) })
	if err == nil {
		t.Fatal("a budget of one byte took a loop; the probe measures nothing")
	}
	m := overBudget.FindStringSubmatch(err.Error())
	if m == nil {
		t.Fatalf("the refusal does not say by how much: %v", err)
	}
	n, perr := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
	if perr != nil {
		t.Fatal(perr)
	}
	d.mapPane.call("SetImageBudget", func() { _ = d.mapPane.m.SetImageBudget(int64(n)) })
}

// TestAZoomAcrossTheBoxesKeepsTheRadar is U2-59: the closer boxes' loops are
// taken in although the budget cannot hold them beside the loops they
// replace, because the replaced loops go first. Taking the new ones in
// first refuses them, and with nothing of theirs held before, the radar
// leaves the map.
func TestAZoomAcrossTheBoxesKeepsTheRadar(t *testing.T) {
	var asked, problems []string
	d := openRadarMap(t, "MRMS", &asked)
	d.cfg.MapProblem = func(p string) { problems = append(problems, p) }
	var frames []MapFrame
	d.cfg.MapFrame = func(f MapFrame) { frames = append(frames, f) }
	r := renamed(radarFeed(t, "MRMS", &asked)(context.Background(), d.mapAsk()), "us-b", 1)
	budgetForOne(t, d, r.Overlays[0])

	m, cmd := d.applyMapRadar(mapRadarMsg{radar: r, region: d.mapPane.region.Name})
	d = settleRadar(t, m.(Dashboard), cmd)
	if _, ok := d.mapPane.radarGiven[RadarLayer+"/us-b"]; !ok || len(d.mapPane.radarGiven) != 1 {
		t.Errorf("the radar holds %v; want the closer box's loop alone", d.radarGivenIDs())
	}
	if st := d.mapPane.m.Loop(); st.Count == 0 {
		t.Error("the map draws no loop after the zoom")
	}
	// AND THE RECORDER SAYS WHICH BOXES A FRAME WAS DRAWN FROM, which is how
	// U2-59 is seen in a live run.
	if len(frames) == 0 || strings.Join(frames[len(frames)-1].Radar, ",") != RadarLayer+"/us-b" {
		t.Errorf("the last frame recorded is drawn from %v; want the closer box's loop", lastRadar(frames))
	}
	if d.radarChipText() == "" || len(problems) != 0 {
		t.Errorf("the chip reads %q and the diagnostics %q; want the source and no refusal", d.radarChipText(), problems)
	}
}

// TestARefusedSwapKeepsTheLoopsItHad is U2-59's other half: where the new
// loops do not fit even alone, the loops the map had are put back, so the
// listener keeps the coarser radar rather than none - and the refusal is the
// diagnostics' (D-124).
func TestARefusedSwapKeepsTheLoopsItHad(t *testing.T) {
	var asked, problems []string
	d := openRadarMap(t, "MRMS", &asked)
	d.cfg.MapProblem = func(p string) { problems = append(problems, p) }
	feed := radarFeed(t, "MRMS", &asked)(context.Background(), d.mapAsk())
	budgetForOne(t, d, renamed(feed, "us-b", 1).Overlays[0])
	big := renamed(feed, "us-b", 3) // three times the frames: over the budget alone

	m, cmd := d.applyMapRadar(mapRadarMsg{radar: big, region: d.mapPane.region.Name})
	d = settleRadar(t, m.(Dashboard), cmd)
	if _, ok := d.mapPane.radarGiven[RadarLayer+"/us-a"]; !ok || len(d.mapPane.radarGiven) != 1 {
		t.Errorf("the radar holds %v; want the loop it had, put back", d.radarGivenIDs())
	}
	if st := d.mapPane.m.Loop(); st.Count == 0 {
		t.Error("the map draws no loop: the one it had was lost")
	}
	if d.radarChipText() == "" {
		t.Error("the chip went with the refused loops; the loop it had is still drawn")
	}
	if len(problems) != 1 || !strings.HasPrefix(problems[0], "Radar") {
		t.Errorf("the diagnostics were told %q; want the one refusal", problems)
	}
}

// lastRadar is the radar loops the last frame recorded names, or none.
func lastRadar(frames []MapFrame) []string {
	if len(frames) == 0 {
		return nil
	}
	return frames[len(frames)-1].Radar
}
