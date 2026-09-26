package tty

// map_radar_test.go — 0.18.0 W8 at the window: the radar is its own command,
// the newest frame first and then the loop (W8.12); the source's chip in the
// upper right, MRMS on green and IEM on orange (D-83); the layer switched off
// takes it away; the playback keys drive the library's loop (D-61, W8.9a);
// the lower 48's source is a Setting (D-83).

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// radarFeed answers with a loop of n frames over southern California from
// the source named; asked records each ask's newestOnly and the IEM choice.
func radarFeed(t *testing.T, source string, asked *[]string) func(context.Context, MapAsk, bool) MapRadar {
	t.Helper()
	png, err := os.ReadFile("testdata/radar-frame.png")
	if err != nil {
		t.Fatal(err)
	}
	return func(_ context.Context, ask MapAsk, newestOnly bool) MapRadar {
		word := "loop"
		if newestOnly {
			word = "newest"
		}
		if ask.RadarIEM {
			word += "+iem"
		}
		*asked = append(*asked, word)
		n := 12
		if newestOnly {
			n = 1
		}
		newest := time.Date(2026, 8, 24, 0, 55, 0, 0, time.UTC)
		var frames []tuimaps.LoopFrame
		for i := n - 1; i >= 0; i-- {
			frames = append(frames, tuimaps.LoopFrame{Valid: newest.Add(-time.Duration(i) * 5 * time.Minute), PNG: png})
		}
		o := tuimaps.RadarImage(RadarLayer+"/us-a", tuimaps.Image{Frames: frames, Provider: tuimaps.ProviderIEM,
			West: -126, South: 23, East: -65, North: 51, Projection: tuimaps.PlateCarree}, newest)
		return MapRadar{Overlays: []tuimaps.Overlay{o}, Source: source}
	}
}

// settleRadar runs the radar's asks and the work they start until nothing
// more is asked.
func settleRadar(t *testing.T, d Dashboard, cmd tea.Cmd) Dashboard {
	t.Helper()
	for range 6 {
		var next []tea.Cmd
		for _, msg := range msgsOf(cmd) {
			if r, ok := msg.(mapRadarMsg); ok {
				m, c := d.applyMapRadar(r)
				d, next = m.(Dashboard), append(next, c)
			}
		}
		if len(next) == 0 {
			break
		}
		cmd = tea.Batch(next...)
	}
	return settleMap(t, d)
}

func openRadarMap(t *testing.T, source string, asked *[]string) Dashboard {
	t.Helper()
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, source, asked), MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	return settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
}

func TestTheRadarComesNewestFirstThenTheLoop(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	if strings.Join(asked, ",") != "newest,loop" {
		t.Fatalf("the radar was asked %v; want the newest frame, then the loop", asked)
	}
	if o, ok := d.mapPane.radarGiven[RadarLayer+"/us-a"]; !ok || len(o.Image.Frames) != 12 {
		t.Fatalf("the loop was not handed in: %v", d.mapPane.radarGiven)
	}
	if st := d.mapPane.m.Loop(); st.Count != 12 || st.Playing {
		t.Errorf("the map opens on %+v; want the loop, stopped on the newest", st)
	}
	if !strings.Contains(stripANSITest(d.mapStatusLine()), "Radar (MRMS)") {
		t.Errorf("the status line does not say the loop: %q", stripANSITest(d.mapStatusLine()))
	}
}

func TestTheSourcesChipIsInTheUpperRight(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(rendering.ResetColorEnabledForTest)
	var asked []string
	for source, ground := range map[string]render.Token{"MRMS": render.MapRadarMRMSBG, "IEM": render.MapRadarIEMBG} {
		d := openRadarMap(t, source, &asked)
		lines := d.mapBodyLines()
		row := lines[1]
		plain := stripANSITest(row)
		if !strings.HasSuffix(strings.TrimRight(plain, " │"), source) && !strings.Contains(plain, " "+source+" ") {
			t.Errorf("%s: the second row is %q, no chip at its right", source, plain)
		}
		if !strings.Contains(row, render.Tok(ground)) {
			t.Errorf("%s: the chip is not on its own ground", source)
		}
		if strings.Contains(stripANSITest(lines[0]), " "+source+" ") {
			t.Errorf("%s: the chip covers the library's top row", source)
		}
	}
}

func TestSwitchingRadarOffTakesItAway(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	d = pressCode(d, 'O', "O")
	var keys []string
	for _, r := range d.overlayRows() {
		keys = append(keys, r.key)
	}
	d.mapPane.menuAt = indexOf(keys, RadarLayer)
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = settleRadar(t, m.(Dashboard), cmd)
	if len(d.mapPane.radarGiven) != 0 || d.radarChipText() != "" {
		t.Errorf("radar off left %v and the chip %q", d.mapPane.radarGiven, d.radarChipText())
	}
}

func TestThePlaybackKeysDriveTheLoop(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "IEM", &asked)
	d = pressCode(d, ',', ",")
	if st := d.mapPane.m.Loop(); st.Index != 10 {
		t.Errorf("',' from the newest went to frame %d, want 10", st.Index)
	}
	d = pressCode(d, '.', ".")
	if d.mapPane.m.Loop().Index != 11 {
		t.Error("'.' did not step on")
	}
	d = pressCode(d, ',', ",")
	d = pressCode(d, 'n', "n")
	if d.mapPane.m.Loop().Index != 11 {
		t.Error("'n' did not return to the newest")
	}
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !m.(Dashboard).mapPane.m.Loop().Playing {
		t.Error("space did not play")
	}
	m, _ = m.(Dashboard).Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if m.(Dashboard).mapPane.m.Loop().Playing {
		t.Error("space again did not stop")
	}
	var help []string
	for _, r := range mapHelpRows(defaultMapKeyMap()) {
		help = append(help, r.keys+" "+r.help)
	}
	if !strings.Contains(strings.Join(help, "\n"), "space, ,, ., n Radar") {
		t.Errorf("Help does not list the playback keys:\n%s", strings.Join(help, "\n"))
	}
}

func TestTheLower48sRadarSourceIsASetting(t *testing.T) {
	d, got := uiDash(t, rowMapRadarSource)
	body, _, _ := d.focusBody(d.opts())
	if text := stripANSITest(strings.Join(body, "\n")); !strings.Contains(text, "Radar -") || !strings.Contains(text, "MRMS") {
		t.Fatalf("the Maps tab has no radar row at MRMS:\n%s", text)
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	if !d.mapRadarIEM || d.radarSourceLabel() != "IEM (lower 48)" || !d.mapAsk().RadarIEM {
		t.Errorf("→ gave %q; want IEM for the lower 48, in the ask", d.radarSourceLabel())
	}
	if back, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft}); back.(Dashboard).mapRadarIEM {
		t.Error("← did not go back to MRMS")
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapRadarSource != "iem" {
		t.Errorf("esc wrote %q, want iem", got.MapRadarSource)
	}
	if mapDash(t, Config{MapRadarSource: "iem"}).mapRadarIEM != true || mapDash(t, Config{}).mapRadarIEM {
		t.Error("the file's word does not open the window as chosen")
	}
	_ = geo.RegionContiguous
}

// TestTheNewestFramesAgeIsAlwaysSaid is W8.8 (FR-5.4, M3): the loop's line
// says how old the newest frame is, and marks it stale past ten minutes -
// never hidden; with MRMS the note under the map says its colours are
// approximate (W8.15a).
func TestTheNewestFramesAgeIsAlwaysSaid(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	d.mapPane.radarNote = "MRMS radar's colours are approximate: its scale is read from its legend."
	d = d.renderMap()
	if line := stripANSITest(d.mapStatusLine()); !strings.Contains(line, "newest 5 min ago") || strings.Contains(line, "stale") {
		t.Errorf("five minutes on, the line is %q", line)
	}
	if !strings.Contains(bodyText(d), "approximate") {
		t.Error("MRMS's note is not under the map")
	}
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 30, 0, 0, time.UTC) }
	d = d.renderMap()
	if line := stripANSITest(d.mapStatusLine()); !strings.Contains(line, "35 min ago, stale") {
		t.Errorf("thirty-five minutes on, the line is %q", line)
	}
}

// TestEveryPlaybackEventDrawsTheFrame is W8.10, guard 2's playback rows: a
// playback key and the radar landing each draw, and the stored frame is the
// library's (W2.5's freshness).
func TestEveryPlaybackEventDrawsTheFrame(t *testing.T) {
	var asked []string
	for name, event := range map[string]func(Dashboard) Dashboard{
		"step back": func(d Dashboard) Dashboard { return pressCode(d, ',', ",") },
		"play": func(d Dashboard) Dashboard {
			m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
			return m.(Dashboard)
		},
		"the radar lands": func(d Dashboard) Dashboard {
			d = d.requestRadar()
			m, _ := d.applyMapRadar(mapRadarMsg{gen: d.mapPane.radarGen, radar: radarFeed(t, "IEM", &asked)(context.Background(), d.mapAsk(), false)})
			return m.(Dashboard)
		},
	} {
		d := openRadarMap(t, "IEM", &asked)
		gen := d.mapPane.gen
		d = event(d)
		if d.mapPane.gen == gen {
			t.Errorf("%s did not draw the map", name)
		}
		assertFresh(t, d, name)
	}
}
