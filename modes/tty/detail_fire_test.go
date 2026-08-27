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
	d.showDetails = true
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
	cold.showDetails = true
	if q := stripANSITest(strings.Join(cold.detailLines(), "\n")); !strings.Contains(q, "FIRE │ Hotspots      fire feed not yet available") {
		t.Fatalf("before any fire feed answers it must say so, never 'none':\n%s", q)
	}
	s3 := snap()
	s3.Locations[0].Fire = snapshot.FireState{AsOf: now}
	m3, _ := m.Update(SnapshotMsg{Snap: s3})
	quiet := m3.(Dashboard)
	quiet.showDetails = true
	if q := stripANSITest(strings.Join(quiet.detailLines(), "\n")); !strings.Contains(q, "FIRE │ Hotspots      none within the fire ring") {
		t.Fatalf("no fire must still be said:\n%s", q)
	}
}
