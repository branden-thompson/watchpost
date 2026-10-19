package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
	"github.com/branden-thompson/go-ionomaps/ionomapstest"
)

// testDay is the tests' first day.
var testDay = time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)

// recordedDays are days as PLAN's fetch left them, written by go-ionomaps'
// ionomapstest (never a service's data): grids at 00:05 and 03:05, foF2 6
// MHz everywhere; four of go-ionomaps' seeded live stations, at their
// places in the mainland US, reading every 15 minutes through the first two
// hours; a solar file of the 30 days before and the day itself, F10.7 100.
func recordedDays(t *testing.T) *ionomapstest.Recorded {
	t.Helper()
	dir := t.TempDir()
	var readings []time.Time
	for m := 0; m <= 120; m += 15 {
		readings = append(readings, testDay.Add(time.Duration(m)*time.Minute))
	}
	solar, err := ionomapstest.Write(dir, ionomapstest.Synthetic{
		Grids: []time.Time{testDay.Add(5 * time.Minute), testDay.Add(3*time.Hour + 5*time.Minute)}, GridFoF2: 6,
		Stations: []ionomapstest.Station{
			{Code: "AL945", Lat: 45.1, Lon: -83.6, FoF2: 6, M3000: 3},
			{Code: "EG931", Lat: 30.5, Lon: -86.5, FoF2: 6, M3000: 3},
			{Code: "IF843", Lat: 43.8, Lon: -112.7, FoF2: 6, M3000: 3},
			{Code: "MHJ45", Lat: 42.6, Lon: -71.5, FoF2: 6, M3000: 3},
		},
		Readings: readings, F107: 100})
	if err != nil {
		t.Fatal(err)
	}
	in, err := ionomapstest.Open(dir, solar)
	if err != nil {
		t.Fatal(err)
	}
	return in
}

// TestEachColumnIsFedAsItSays runs the replay through go-ionomaps: the grids
// it serves are the background of every column but the climatology's, the
// replies are read and assimilated in the shipped build's column and no
// other, nothing is left unread - and each scenario is answered with a
// status for its band.
func TestEachColumnIsFedAsItSays(t *testing.T) {
	in := recordedDays(t)
	at := []time.Time{testDay.Add(35 * time.Minute)}
	var snaps [columns]map[time.Time]ionomaps.Snapshot
	for c := range columns {
		s, facts, err := snapshots(in, c, at)
		if err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		f := facts[at[0]]
		if c == shipped && f.Stations < 3 {
			t.Errorf("the shipped build assimilated %d of the four stations' readings; want them read through the replay", f.Stations)
		}
		if c != shipped && f.Stations != 0 || f.Grid != (c != climatology) || f.Unread != 0 {
			t.Errorf("%s: %+v; want no station but in the shipped build, a grid but in the climatology, nothing unread", c, f)
		}
		if f.F107 != 100 {
			t.Errorf("%s: F10.7 mean %v; want the solar file's 100", c, f.F107)
		}
		snaps[c] = s
	}
	set := []scenario{
		{Metric: m1, At: at[0], Band: ionomaps.Band40m, Origin: tower, Target: rxNorth, Km: 100, Outcome: open},
		{Metric: m4, At: at[0], Band: ionomaps.Band20m, Origin: rxNorth, Target: frankfurt, Km: 9000, Outcome: unheard},
	}
	res, err := scoreSet(set, snaps, reach{SkyKm: defaultSkyKm, ServedKm: defaultServedKm, RegionKm: defaultRegionKm, NearKm: defaultNearKm})
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := res.tallies[base{m4, ionomaps.Band20m}][shipped].total(); n != 1 || len(res.answers) != 2 {
		t.Errorf("the set answered: %+v", res)
	}
	var b bytes.Buffer
	d := newDrawer(10, 1, true)
	for _, s := range set {
		d.offer(s)
	}
	facts := [columns]map[time.Time]hourFacts{}
	if err := report(&b, reportOf{flags: scoreFlags{seed: 1, k: 10, strata: strataBandHour}, days: []time.Time{testDay}, set: set, counts: d.counts, res: res, facts: facts, rev: "test"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## M1", "## M4", "| 40m | 1 (1 / 0 / 0) |", "| 20m | 1 (0 / 0 / 1) |", "Against the target (D-99, judged on the point estimate", "## The replay"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the table lacks %q:\n%s", want, b.String())
		}
	}
}
