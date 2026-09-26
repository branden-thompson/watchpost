package tty

// map_regions_test.go — 0.18.0 D-77 (UAT-1 U1-40): the listener reaches every
// region the station's APIs cover - a number snaps to each, and panning past a
// region's edge moves on to its neighbour in the HUM LEAD's arrangement. The
// map stays bound to one region at a time (D-28).

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// regionShown is the region the map is bound to, and whether the map's
// centre lies in it.
func regionShown(d Dashboard) (string, bool) {
	c, _ := d.mapPane.m.Centre()
	return d.mapPane.region.Name, d.mapPane.region.Contains(c.Lat, c.Lon)
}

func TestTheRegionKeysSnapToEachRegion(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	for n, want := range []string{geo.RegionContiguous, geo.RegionAlaska, geo.RegionHawaii, geo.RegionCaribbean, geo.RegionSamoa, geo.RegionMarianas} {
		key := string(rune('1' + n))
		d = pressCode(d, rune('1'+n), key)
		if got, inside := regionShown(d); got != want || !inside {
			t.Errorf("%s shows %q (centre inside: %v), want %s", key, got, inside, want)
		}
	}
	if d.mapPane.flash != mapRegionActs[5] {
		t.Errorf("6 blinked %q, not its chip", d.mapPane.flash)
	}
}

// panUntilRegion presses a pan key until the region changes, up to a bound.
func panUntilRegion(t *testing.T, d Dashboard, code rune) Dashboard {
	t.Helper()
	was := d.mapPane.region.Name
	for range 80 {
		m, _ := d.Update(tea.KeyPressMsg{Code: code})
		d = m.(Dashboard)
		if d.mapPane.region.Name != was {
			return d
		}
	}
	t.Fatalf("eighty presses never left %s", was)
	return d
}

func TestPanningPastAnEdgeMovesToTheNeighbour(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	if got, _ := regionShown(panUntilRegion(t, d, tea.KeyRight)); got != geo.RegionCaribbean {
		t.Errorf("from the state scale, east past the coast is %q", got) // the listener's own path: pan until the edge
	}
	d = pressCode(d, '1', "1") // the whole region, so each edge is a press or two off
	for _, step := range []struct {
		key  rune
		want string
	}{
		{tea.KeyRight, geo.RegionCaribbean},
		{tea.KeyLeft, geo.RegionContiguous},
		{tea.KeyLeft, geo.RegionHawaii},
		{tea.KeyLeft, geo.RegionSamoa},
		{tea.KeyLeft, geo.RegionMarianas},
		{tea.KeyRight, geo.RegionSamoa},
		{tea.KeyUp, geo.RegionSamoa}, // no neighbour north of Samoa: the region stays
	} {
		if step.key == tea.KeyUp {
			for range 20 {
				m, _ := d.Update(tea.KeyPressMsg{Code: step.key})
				d = m.(Dashboard)
			}
		} else {
			d = panUntilRegion(t, d, step.key)
		}
		if got, inside := regionShown(d); got != step.want || !inside {
			t.Fatalf("panning %v reached %q (centre inside: %v), want %s", step.key, got, inside, step.want)
		}
	}
	d = pressCode(d, '1', "1")
	d = panUntilRegion(t, d, tea.KeyUp)
	if got, _ := regionShown(d); got != geo.RegionAlaska {
		t.Errorf("north of the contiguous United States is %q, want Alaska", got)
	}
	d = panUntilRegion(t, d, tea.KeyDown)
	if got, _ := regionShown(d); got != geo.RegionContiguous {
		t.Errorf("south of Alaska is %q, want the contiguous United States", got)
	}
}

func TestTheRegionKeysAreShownWithTheControls(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	box := stripANSITest(strings.Join(d.controlsBox(), "\n"))
	for _, want := range []string{"1", "6", "region"} {
		if !strings.Contains(box, want) {
			t.Errorf("the controls box does not show %q:\n%s", want, box)
		}
	}
	var help []string
	for _, r := range mapHelpRows(defaultMapKeyMap()) {
		help = append(help, r.keys+" "+r.help)
	}
	if text := strings.Join(help, "\n"); !strings.Contains(text, "1, 2, 3 Region: US, Alaska, Hawaii") || !strings.Contains(text, "4, 5, 6 Region: Caribbean, Samoa, Guam") {
		t.Errorf("Help does not list the region keys:\n%s", text)
	}
}
