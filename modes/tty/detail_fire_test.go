package tty

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestDetailFireSectionAlwaysPresent(t *testing.T) {
	// B5 (HUM LEAD 2026-08-25): fire is another alert kind — a FIRE section
	// in the detail modal: hotspots nearest first with bearing, distance,
	// strength (bold at the threshold), satellite and age; named incidents
	// with acres and containment; the fire-weather alert when one is active.
	// With nothing burning it still says so.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	loc := &s2.Locations[0]
	now := time.Now()
	loc.Fire = snapshot.FireState{
		AsOf: now,
		Hotspots: []snapshot.Hotspot{
			{Lat: loc.Lat + 0.09, Lon: loc.Lon, DetectedAt: now.Add(-2 * time.Hour), FRPMW: f64(62), DistanceKm: f64(10), Source: snapshot.SourceInfo{Provider: "hms", ModelOrStation: "GOES-WEST"}},
			{Lat: loc.Lat, Lon: loc.Lon + 0.2, DetectedAt: now.Add(-40 * time.Minute), FRPMW: f64(8), DistanceKm: f64(18), Source: snapshot.SourceInfo{Provider: "hms", ModelOrStation: "NOAA-20"}},
		},
		Incidents: []snapshot.Incident{{Name: "Timber", Acres: f64(12915), PercentContained: f64(26), Discovered: now.Add(-72 * time.Hour), Source: snapshot.SourceInfo{Provider: "wfigs", DistanceKm: f64(30)}}},
	}
	loc.Alerts = append(loc.Alerts, snapshot.Alert{ID: "rfw", Event: "Red Flag Warning", Severity: "severe", Expires: now.Add(6 * time.Hour)})
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.modal = modalDetails
	raw := strings.Join(d.detailLines(), "\n")
	joined := stripANSITest(raw)
	for _, want := range []string{
		"FIRE │ Hotspots      2 hotspots nearby",
		"◆ 6 mi N    62 MW · GOES-WEST",
		"2h 00m",
		"◆ 11 mi E   8 MW · NOAA-20",
		"Timber      19 mi · 12,915 ac",
		"26% contained",
		"Fire Wx       Red Flag Warning",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("FIRE section missing %q:\n%s", want, joined)
		}
	}
	if !strings.Contains(raw, render.Tint("62 MW", "1;"+render.Tok(render.FireMark))) {
		t.Fatalf("62 MW must read bold at the 50 MW threshold:\n%q", raw)
	}
	if strings.Contains(raw, render.Tint("8 MW", "1;"+render.Tok(render.FireMark))) {
		t.Fatal("8 MW must not read bold")
	}
	cold := m.(Dashboard)
	cold.modal = modalDetails
	if q := stripANSITest(strings.Join(cold.detailLines(), "\n")); !strings.Contains(q, "FIRE │ Hotspots      fire feed not yet available") {
		t.Fatalf("before any fire feed answers it must say so, never 'none':\n%s", q)
	}
	s3 := snap()
	s3.Locations[0].Fire = snapshot.FireState{AsOf: now}
	m3, _ := m.Update(SnapshotMsg{Snap: s3})
	quiet := m3.(Dashboard)
	quiet.modal = modalDetails
	if q := stripANSITest(strings.Join(quiet.detailLines(), "\n")); !strings.Contains(q, "FIRE │ Hotspots      none within the fire ring") {
		t.Fatalf("no fire must still be said:\n%s", q)
	}
}

// THE DETAIL SHOWS EVERY NAMED FIRE THE ROW COUNTS (UAT 2026-09-07).
//
// The row wears n◆ from len(Incidents) and this list broke at three with no
// word about the rest, so a location with five named fires read "5◆" on the
// dashboard and listed three in the one place the app sends people for more
// detail. The spoken report, meanwhile, names ALL of them — so the same
// location had three different answers to "which fires are near me", and the
// most complete one was the one you cannot re-read.
//
// HUM LEAD, 2026-09-07: "we direct user to the location detail for 'more
// details' so location detail modal needs to show all 5."
//
// Hotspots keep their cap and their "… and N more": there can be 300 of them
// (snapshot.MaxHotspots) and they have no names to tell apart. Named incidents
// are bounded by the incident radius and are exactly what a listener is asking
// about.
func TestTheFireDetailListsEveryNamedIncident(t *testing.T) {
	names := []string{"MUTUAL AID", "ORTEGA", "SC/PMQ", "BRENGEL", "FUR CREEK"}
	loc := &snapshot.Location{Label: "Oceanside, CA", Fire: snapshot.FireState{AsOf: time.Now()}}
	for i, n := range names {
		km := float64(18 + i)
		loc.Fire.Incidents = append(loc.Fire.Incidents, snapshot.Incident{
			Name: n, Source: snapshot.SourceInfo{DistanceKm: &km},
		})
	}

	got := stripANSITest(strings.Join(fireRows(render.Opts{Width: 100}, loc, time.Now(), 50), "\n"))
	for _, n := range names {
		if !strings.Contains(got, ellipsizeName(n)) {
			t.Errorf("%q is one of the %d fires the row counts and the detail does not list it:\n%s",
				n, len(names), got)
		}
	}
	// AND THE COUNT ON THE ROW IS THE COUNT IN THE LIST, which is the thing the
	// two surfaces disagreed about.
	if got, want := fireCount(loc.Fire), len(names); got != want {
		t.Errorf("the row wears %d◆ for %d incidents", got, want)
	}
}

// ellipsizeName is how the row draws a name, so the assertion compares what is
// actually on screen rather than the name the fixture used.
func ellipsizeName(n string) string { return ellipsize(n, 11, false) }
