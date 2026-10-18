package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// The tests' recorded stations: four of go-ionomaps' seeded live stations,
// in the mainland US, where they are as GIRO's location line says it.
var testStations = []struct {
	code     string
	lat, lon float64 // the longitude east, 0 to 360, as GIRO writes it
}{{"AL945", 45.1, 276.4}, {"EG931", 30.5, 273.5}, {"IF843", 43.8, 247.3}, {"MHJ45", 42.6, 288.5}}

// testDay is the tests' first day.
var testDay = time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)

// writeRecorded writes a folder as PLAN's fetch left it: two grids, each
// station's reply with readings every 15 minutes through the first two
// hours, and a solar file of the 30 days before and the day itself.
func writeRecorded(t *testing.T) (dir, solar string) {
	t.Helper()
	dir = t.TempDir()
	for _, name := range []string{"glotec_icao_20260929T000500Z.geojson", "glotec_icao_20260929T030500Z.geojson"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o600); err != nil { // not a grid: go-ionomaps reads it as a changed format
			t.Fatal(err)
		}
	}
	for _, s := range testStations {
		var b strings.Builder
		fmt.Fprintf(&b, "# Global Ionospheric Radio Observatory (GIRO)\n# Location: GEO ( %.1f N   %.1f E ), URSI-Code %s, TEST\n", s.lat, s.lon, s.code)
		b.WriteString("# Requested by unknown guest\n# Time                    CS   foF2 QD MUF(D) QD M(D) QD   hmF2 QD\n")
		for m := 0; m <= 120; m += 15 {
			at := testDay.Add(time.Duration(m) * time.Minute).Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(&b, "%s  90  6.000 // 18.000 // 3.00 //  250.0 //\n", at)
		}
		if err := os.WriteFile(filepath.Join(dir, "fc_"+s.code+".txt"), []byte(b.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var b strings.Builder
	b.WriteString(":Product: Daily Solar Data            DSD.txt\n#  Date     10.7cm Number\n")
	for d := -30; d <= 0; d++ {
		fmt.Fprintf(&b, "%s  100     55      110      0    -999      *   0  0  0  2  0  0  0\n", testDay.AddDate(0, 0, d).Format("2006 01 02"))
	}
	solar = filepath.Join(dir, "dsi.txt")
	if err := os.WriteFile(solar, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, solar
}

// TestTheReplayServesNothingAfterItsClock is the replay's one promise: at
// its clock it answers as the services would have then - the grids valid by
// then and no later, the solar rows of the days before, the readings in the
// window asked and not after the clock, in the columns go-ionomaps asks,
// without the line naming the requester.
func TestTheReplayServesNothingAfterItsClock(t *testing.T) {
	dir, solar := writeRecorded(t)
	in, err := loadRecorded(dir, solar)
	if err != nil {
		t.Fatal(err)
	}
	r := &replay{in: in, grids: true, now: testDay.Add(2*time.Hour + 35*time.Minute)}
	ctx := context.Background()
	idx, _ := r.Fetch(ctx, glotecIndex, ionomaps.Validators{})
	if !strings.Contains(string(idx.Body), "T000500Z") || strings.Contains(string(idx.Body), "T030500Z") {
		t.Errorf("the index at 02:35 names other than the 00:05 grid:\n%s", idx.Body)
	}
	if g, _ := r.Fetch(ctx, glotecIndex+"glotec_icao_20260929T030500Z.geojson", ionomaps.Validators{}); g.Status != replayMissing {
		t.Errorf("the 03:05 grid at 02:35: %d; want missing", g.Status)
	}
	r.now = testDay.Add(3*time.Hour + 35*time.Minute)
	if g, _ := r.Fetch(ctx, glotecIndex+"glotec_icao_20260929T030500Z.geojson", ionomaps.Validators{}); g.Status != replayOK {
		t.Errorf("the 03:05 grid at 03:35: %d; want served", g.Status)
	}
	sol, _ := r.Fetch(ctx, solarFile, ionomaps.Validators{})
	if !strings.Contains(string(sol.Body), "2026 09 28") || strings.Contains(string(sol.Body), "2026 09 29") || !strings.Contains(string(sol.Body), ":Product:") {
		t.Errorf("the solar file on the 29th holds the 29th's row, or lacks the 28th's or its header:\n%s", sol.Body)
	}
	r.now = testDay.Add(50 * time.Minute)
	u := fastChar + "ursiCode=AL945&charName=foF2%2CM%28D%29&DMUF=3000&fromDate=2026%2F09%2F29+00%3A10%3A00&toDate=2026%2F09%2F29+00%3A50%3A00"
	st, err := r.Fetch(ctx, u, ionomaps.Validators{})
	if err != nil {
		t.Fatal(err)
	}
	var rows []string
	for line := range strings.Lines(string(st.Body)) {
		if !strings.HasPrefix(line, "#") {
			rows = append(rows, strings.Join(strings.Fields(line), " "))
		}
	}
	want := []string{"2026-09-29T00:15:00.000Z 90 6.000 // 3.00 //", "2026-09-29T00:30:00.000Z 90 6.000 // 3.00 //", "2026-09-29T00:45:00.000Z 90 6.000 // 3.00 //"}
	if strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Errorf("AL945 from 00:10 to 00:50 at 00:50:\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(want, "\n"))
	}
	if strings.Contains(string(st.Body), "Requested by") || !strings.Contains(string(st.Body), "URSI-Code AL945") {
		t.Errorf("the reply keeps the requester's line, or lost the station's:\n%s", st.Body)
	}
	r.now = testDay.Add(20 * time.Minute)
	if st, _ := r.Fetch(ctx, u, ionomaps.Validators{}); strings.Contains(string(st.Body), "T00:30:00") {
		t.Errorf("at 00:20 the reply holds a reading from 00:30:\n%s", st.Body)
	}
}

// TestTheReplayAnswersOnlyWhatItHolds is the replay's edges: the
// climatology's column is never given a grid; D-RAP, the scales and a
// station not recorded are missing; any other address fails the run.
func TestTheReplayAnswersOnlyWhatItHolds(t *testing.T) {
	dir, solar := writeRecorded(t)
	in, err := loadRecorded(dir, solar)
	if err != nil {
		t.Fatal(err)
	}
	r := &replay{in: in, grids: false, now: testDay.Add(4 * time.Hour)}
	ctx := context.Background()
	for _, u := range []string{glotecIndex, glotecIndex + "glotec_icao_20260929T000500Z.geojson", drapTable, scalesFile,
		fastChar + "ursiCode=ZZ999&fromDate=2026%2F09%2F29+00%3A00%3A00&toDate=2026%2F09%2F29+01%3A00%3A00"} {
		if got, err := r.Fetch(ctx, u, ionomaps.Validators{}); err != nil || got.Status != replayMissing {
			t.Errorf("%s: %d, %v; want missing", u, got.Status, err)
		}
	}
	if _, err := r.Fetch(ctx, "https://example.org/other", ionomaps.Validators{}); !errors.Is(err, errUnknownAsk) || len(r.unknown) != 1 {
		t.Errorf("an unknown address: %v, recorded %v", err, r.unknown)
	}
	if _, err := loadRecorded(t.TempDir(), solar); !errors.Is(err, errNoInputs) {
		t.Errorf("an empty folder: %v; want errNoInputs", err)
	}
}

// TestEachColumnIsFedAsItSays runs the replay through go-ionomaps: the
// replies it serves are read and assimilated in the shipped build's column,
// no other column is given a station's reading, the climatology's no grid -
// and each scenario is answered with a status for its band.
func TestEachColumnIsFedAsItSays(t *testing.T) {
	dir, solar := writeRecorded(t)
	in, err := loadRecorded(dir, solar)
	if err != nil {
		t.Fatal(err)
	}
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
		if c != shipped && f.Stations != 0 || f.Grid {
			t.Errorf("%s: %+v; want no station, and no grid (the test's grids are not grids)", c, f)
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
	for _, want := range []string{"## M1", "## M4", "| 40m | 1 (1 / 0 / 0) |", "| 20m | 1 (0 / 0 / 1) |", "Against the target (D-99)", "## The replay"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the table lacks %q:\n%s", want, b.String())
		}
	}
}
