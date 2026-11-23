package tty

// console_fire_test.go — the console reads the operator's fire threshold, the one
// Observer reads (0.19.6 D-54, D-55).

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/branden-thompson/watchpost/modes/tty/internal/frame"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// TestTheConsoleReadsTheOperatorsFireThreshold.
//
// ONE PLACE, ONE ANSWER ABOUT WHETHER A FIRE IS BURNING HARD. A console
// threshold of its own — Observer's DEFAULT 50, stated as a constant — would
// honour an operator's `bold_frp_mw` in [fire] on the watchlist and ignore it
// on the console, so one location could show a bold hotspot on one surface and
// a plain one on the other. Two carriers of one fact.
//
// IT IS INHERITED THE WAY `ascii` AND `version` ARE — copied by `NewRouter`
// from the Dashboard the Router is built over, RESOLVED (`d.fireBoldMW()`)
// rather than raw, so the default lives in exactly one place and the console
// cannot re-derive it differently. `cfg` is written once at construction and
// never reassigned, so there is nothing later to follow.
func TestTheConsoleReadsTheOperatorsFireThreshold(t *testing.T) {
	frp := 100.0
	loc := ttytest.PoolLoc()
	loc.Fire.Hotspots = []snapshot.Hotspot{{FRPMW: &frp}}
	ref := snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon}

	// 100 MW is over the default 50 and under an operator's 200: the one
	// fixture that can tell the two thresholds apart.
	rendering.SetColorEnabledForTest(true) // the bold is the mark's tint (0.19.6 D-54)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
	// A MARK IS MATCHED UP TO ITS GLYPH: the frame re-arms every reset to its
	// own base tone, so the tint's closing reset is not in it.
	fire := render.Opts{ASCII: true}.Glyphs().Fire
	boldMark := strings.TrimSuffix(render.Tint(fire, "1;"+render.Tok(render.FireMark)), "\x1b[0m")
	plainMark := strings.TrimSuffix(render.Tint(fire, render.Tok(render.FireMark)), "\x1b[0m")
	for _, tc := range []struct {
		name    string
		set     float64
		wantHot bool
	}{
		{"the default, unset", 0, true},
		{"the operator raised it above this fire", 200, false},
		{"the operator lowered it below this fire", 25, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewDashboard(Config{FireBoldMW: tc.set, ASCII: true})
			if err != nil {
				t.Fatal(err)
			}
			r := NewRouter(d)
			b := r.broadcaster
			b, _ = b.Update(tea.WindowSizeMsg{Width: consoleW, Height: consoleH})
			b, _ = b.Update(StationAreaMsg{
				Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
				RadiusMi:    50, Pool: []snapshot.LocationRef{ref}})
			b, _ = b.WithRecent(&snapshot.Snapshot{Locations: []snapshot.Location{*loc}})

			// AS THE OPERATOR SEES IT (0.19.6 D-54): the pool row's fire mark is
			// drawn bold while the row is hot and plain while it is not.
			got := b.View().Content
			if !strings.Contains(ttytest.StripANSI(got), loc.Label) {
				t.Fatalf("the pool drew no row for its one candidate")
			}
			hot, plain := strings.Contains(got, boldMark), strings.Contains(got, plainMark)
			if hot == plain {
				t.Fatalf("the pool row's fire mark is drawn bold %v and plain %v; one candidate draws one", hot, plain)
			}
			if hot != tc.wantHot {
				t.Errorf("a %g MW hotspot with the threshold at %g: the console says hot=%v, want %v",
					frp, tc.set, hot, tc.wantHot)
			}
			// AND THE TWO SURFACES AGREE, which is the rule a constant of its own
			// breaks: Observer's answer is the row it builds (0.19.6 D-55).
			if want := frame.WeatherRow(loc, d.fireBoldMW(), 0).FireHot; hot != want {
				t.Errorf("the console says hot=%v and Observer says %v for one place",
					hot, want)
			}
		})
	}
}
