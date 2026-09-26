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

// TestTheEdgeShowsItsNeighbourBeforeCrossing is D-81 (UAT-1 U1-44): a press
// the edge holds still does not cross - it shows a chip naming the region
// beyond, at that edge - and the next press the same way crosses; a press
// another way takes the chip away and pans as normal.
func TestTheEdgeShowsItsNeighbourBeforeCrossing(t *testing.T) {
	d := openMap(t, Config{MapDescription: "off"}, 133, 44)
	d = pressCode(d, '1', "1")
	right := tea.KeyPressMsg{Code: tea.KeyRight}
	d = untilChip(t, d, tea.KeyRight)
	if got, _ := regionShown(d); got != geo.RegionContiguous {
		t.Fatalf("the first press at the edge crossed to %s", got)
	}
	if text := bodyText(d); !strings.Contains(text, "US CARIBBEAN →") {
		t.Errorf("no chip names the region beyond:\n%s", text)
	}
	m, _ := d.Update(right)
	if got, _ := regionShown(m.(Dashboard)); got != geo.RegionCaribbean {
		t.Errorf("the second press went to %s, want the Caribbean", got)
	}
	m, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyLeft}) // another way, from the chip
	d = m.(Dashboard)
	if strings.Contains(bodyText(d), "US CARIBBEAN") || d.mapPane.edgeShown {
		t.Error("a press another way left the chip")
	}
	if got, _ := regionShown(d); got != geo.RegionContiguous {
		t.Errorf("a press another way changed the region to %s", got)
	}
	d = untilChip(t, d, tea.KeyRight)
	m, _ = d.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	if got, _ := regionShown(m.(Dashboard)); got != geo.RegionContiguous || m.(Dashboard).mapPane.edgeShown {
		t.Error("a key other than a pan left the chip standing, or crossed")
	}
	for _, c := range []struct {
		n    rune
		dir  rune
		want string
	}{{'3', tea.KeyLeft, "← AMERICAN SAMOA"}, {'1', tea.KeyUp, "↑ ALASKA"}, {'2', tea.KeyDown, "↓ CONTINENTAL US"}} {
		d = untilChip(t, pressCode(d, c.n, string(c.n)), c.dir)
		if text := bodyText(d); !strings.Contains(text, c.want) {
			t.Errorf("region %c, %v: no %q chip", c.n, c.dir, c.want)
		}
	}
}

// untilChip presses a pan key until the edge's chip shows, the region
// unchanged, up to a bound.
func untilChip(t *testing.T, d Dashboard, code rune) Dashboard {
	t.Helper()
	was := d.mapPane.region.Name
	for range 80 {
		m, _ := d.Update(tea.KeyPressMsg{Code: code})
		d = m.(Dashboard)
		if d.mapPane.region.Name != was {
			t.Fatalf("a press crossed from %s with no chip shown first", was)
		}
		if d.mapPane.edgeShown {
			return d
		}
	}
	t.Fatalf("eighty presses never reached the edge of %s", was)
	return d
}

// TestAChipCrossesOnlyTheWayItPoints is D-81's second press: the chip shown
// at one edge does not let a press at another held edge cross - that press
// shows its own chip first.
func TestAChipCrossesOnlyTheWayItPoints(t *testing.T) {
	d := openMap(t, Config{MapDescription: "off"}, 133, 44)
	d = untilChip(t, pressCode(d, '5', "5"), tea.KeyRight) // Samoa: Hawaii to the east
	d = untilChip(t, d, tea.KeyLeft)                       // then its west edge: Guam
	if got, _ := regionShown(d); got != geo.RegionSamoa {
		t.Fatalf("a press west under the east chip crossed to %s", got)
	}
	if text := bodyText(d); !strings.Contains(text, "← GUAM") {
		t.Errorf("the west edge's chip does not name Guam:\n%s", text)
	}
}
