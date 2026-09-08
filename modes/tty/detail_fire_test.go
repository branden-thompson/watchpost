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
	// THE TWO RINGS ARE CONFIGURED, because the section states each one beside
	// the list it admits and a test with neither set would never draw them.
	d.cfg.FireRadiusKm, d.cfg.FireIncidentRadiusKm = 25, 50
	raw := strings.Join(d.detailLines(), "\n")
	joined := stripANSITest(raw)
	for _, want := range []string{
		// EACH LIST NAMES ITS OWN RING (UAT 2026-09-07). One heading over two
		// feeds is what made "none within the fire ring" read as a claim about
		// the named fires printed under it.
		"Hotspots  - Radius: 16 mi",
		"Incidents - Radius: 31 mi",
		// A hotspot is a satellite pixel: where, how hard, which bird, when,
		// how sure. It has no name and no acres.
		"62 MW",
		"GOES-WEST",
		"2h 00m",
		"8 MW",
		"NOAA-20",
		// A named fire has a name, a size and a containment, and no radiative
		// power.
		"Timber",
		"12,915 acres",
		"26% contained",
		"Fire Wx       Red Flag Warning",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("FIRE section missing %q:\n%s", want, joined)
		}
	}
	// THE BOLD IS ON THE CELL, not on the words: the table styles a padded cell,
	// so the assertion asks which LINE carries the emphasis rather than
	// rebuilding the exact string the kit produced.
	// The kit expands a token into a full SGR ("208" -> "38;5;208"), so the
	// assertion asks for the BOLD attribute rather than rebuilding the escape.
	const bold = "\x1b[1;"
	for _, l := range strings.Split(raw, "\n") {
		plain := stripANSITest(l)
		switch {
		case strings.Contains(plain, "62 MW") && !strings.Contains(l, bold):
			t.Fatalf("62 MW must read bold at the 50 MW threshold:\n%q", l)
		case strings.Contains(plain, "8 MW") && !strings.Contains(plain, "62 MW") && strings.Contains(l, bold):
			t.Fatalf("8 MW must not read bold:\n%q", l)
		}
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
	// THE QUIET ANSWER IS SAID TWICE, ONCE PER RING (UAT 2026-09-07). A single
	// "none" over two lists is what let a listener read it as covering the
	// named fires printed underneath.
	q := stripANSITest(strings.Join(quiet.detailLines(), "\n"))
	if n := strings.Count(q, "none within this radius"); n != 2 {
		t.Fatalf("each ring says its own none; got %d:\n%s", n, q)
	}
	if !strings.Contains(q, "Hotspots") || !strings.Contains(q, "Incidents") {
		t.Fatalf("both lists are named even when both are empty:\n%s", q)
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

	got := stripANSITest(strings.Join(fireRows(render.Opts{Width: 100}, loc, time.Now(), 50, 25, 50, 65), "\n"))
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

// A LONG NAME COSTS THE AGE COLUMN, NOT THE CONTAINMENT (HUM LEAD, 2026-09-07).
//
// The name is never shortened — that is the ruling this table was built on — so
// an unusually long one has to come out of something. It used to come out of
// the RIGHT EDGE: the row ran past the section and was clamped, silently, which
// took the containment and the age together and left no mark saying so.
//
// "Age can be truncatable - containment is more important." So the age is the
// column that volunteers, and it goes WHOLE rather than shrinking to "...",
// because a cell too narrow to hold a value is a cell shaped like one that says
// nothing.
//
// THERE IS STILL A LENGTH THIS CANNOT SURVIVE. With the age gone, a name past
// about twenty-four cells leaves the row wider than the section and the
// containment is clamped after all. Wrapping the name onto a second line is the
// only thing that would fix that, and it is not built.
func TestALongIncidentNameCostsTheAgeAndNotTheContainment(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	now := time.Now()
	f := func(v float64) *float64 { return &v }
	loc := &snapshot.Location{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.38,
		Fire: snapshot.FireState{AsOf: now, Incidents: []snapshot.Incident{{
			Name: "SAN LUIS REY COMPLEX", Lat: 33.19, Lon: -117.24,
			Acres: f(18000), PercentContained: f(35), Discovered: now.Add(-200 * time.Hour),
			Source: snapshot.SourceInfo{DistanceKm: f(12)},
		}}}}

	got := stripANSITest(strings.Join(fireRows(render.Opts{Width: 85}, loc, now, 50, 25, 50, 65), "\n"))
	if !strings.Contains(got, "SAN LUIS REY COMPLEX") {
		t.Errorf("the name was shortened, which is the one thing this table does not do:\n%s", got)
	}
	if !strings.Contains(got, "35% contained") {
		t.Errorf("the containment was lost to the name; it is the more important of the two:\n%s", got)
	}
	if !strings.Contains(got, "18,000 acres") {
		t.Errorf("the size was lost:\n%s", got)
	}
	// THE AGE WENT WHOLE, not to a stub.
	if strings.Contains(got, "8d") {
		t.Errorf("the age survived in some form; it was the column that volunteered:\n%s", got)
	}
	if strings.Contains(got, "...") || strings.Contains(got, "…") {
		t.Errorf("a column was cut to a tail instead of dropped:\n%s", got)
	}
}
