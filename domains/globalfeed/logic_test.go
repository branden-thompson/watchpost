package globalfeed

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestVerbAndArticleByClass(t *testing.T) {
	rows := []struct {
		class Class
		typ   string
		verb  string
		art   string
	}{
		{ClassQuake, "Earthquake", "recorded", "An"},
		{ClassTropical, "Hurricane", "reported", "A"},
		{ClassTropical, "Tropical Storm", "reported", "A"},
		{ClassSevereWx, "Tornado Warning", "declared", "A"},
		{ClassSevereWx, "Extreme Wind Warning", "declared", "An"},
	}
	for _, r := range rows {
		e := Event{Class: r.class, Type: r.typ}
		if e.Verb() != r.verb {
			t.Errorf("%s verb = %q, want %q", r.typ, e.Verb(), r.verb)
		}
		if e.Article() != r.art {
			t.Errorf("%s article = %q, want %q", r.typ, e.Article(), r.art)
		}
	}
}

func TestSeverityMaps(t *testing.T) {
	if quakeSeverity(6.6, false) != SevRed || quakeSeverity(4.0, true) != SevRed {
		t.Fatal("a great quake or any tsunami is red")
	}
	if quakeSeverity(5.6, false) != SevOrange || quakeSeverity(5.0, false) != SevYellow {
		t.Fatal("5.5–6.5 orange, below yellow")
	}
	if _, s, ok := tropicalClass("HU"); !ok || s != SevRed {
		t.Fatal("a hurricane is red")
	}
	if _, _, ok := tropicalClass("EX"); ok {
		t.Fatal("a post-tropical low is not carried")
	}
	if severeSeverity("Tornado Warning") != SevRed || severeSeverity("Tornado Watch") != SevYellow {
		t.Fatal("tornado warning red, watch yellow")
	}
}

func TestSortMostRecentThenSevere(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	evs := []Event{
		{ID: "old", At: now.Add(-3 * time.Hour), Severity: SevRed},
		{ID: "newYellow", At: now, Severity: SevYellow},
		{ID: "newRed", At: now, Severity: SevRed}, // same instant as newYellow → severity breaks the tie
	}
	Sort(evs)
	if evs[0].ID != "newRed" || evs[1].ID != "newYellow" || evs[2].ID != "old" {
		t.Fatalf("most-recent-first, then most-severe: %v", ids(evs))
	}
}

func TestMergeDedupsAndSplitsNewVsSeen(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	fetched := []Event{
		{ID: "a", At: now, Severity: SevRed},
		{ID: "a", At: now, Severity: SevRed}, // duplicate id — one entry
		{ID: "b", At: now.Add(-time.Hour), Severity: SevOrange},
		{ID: "", At: now}, // no id — dropped
	}
	seen := map[string]bool{"b": true}
	stack, fresh := Merge(fetched, seen)
	if len(stack) != 2 || stack[0].ID != "a" {
		t.Fatalf("deduped, most-recent-first: %v", ids(stack))
	}
	if len(fresh) != 1 || fresh[0].ID != "a" {
		t.Fatalf("only the unseen id is new: %v", ids(fresh))
	}
}

func TestActiveDropsExpiredKeepsWindowlessAndUnexpired(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	evs := []Event{
		{ID: "expired", Until: now.Add(-time.Minute)},    // window closed → dropped
		{ID: "active", Until: now.Add(30 * time.Minute)}, // still active → kept
		{ID: "quake"}, // no Until (zero) → kept
	}
	got := Active(evs, now)
	if len(got) != 2 || got[0].ID != "active" || got[1].ID != "quake" {
		t.Fatalf("expired dropped, active + windowless kept: %v", ids(got))
	}
	if len(evs) != 3 {
		t.Fatal("Active must not mutate its input")
	}
}

func TestGeoPointHandlesPointPolygonAndEmpty(t *testing.T) {
	lat, lon, ok := geoPoint(json.RawMessage(`[-117.2, 32.9]`)) // Point
	if !ok || lat != 32.9 || lon != -117.2 {
		t.Fatalf("Point: %v %v %v", lat, lon, ok)
	}
	lat, lon, ok = geoPoint(json.RawMessage(`[[[-95.4, 29.7],[-95.3, 29.8]]]`)) // Polygon → first vertex
	if !ok || lat != 29.7 || lon != -95.4 {
		t.Fatalf("Polygon first vertex: %v %v %v", lat, lon, ok)
	}
	if _, _, ok := geoPoint(json.RawMessage(``)); ok {
		t.Fatal("absent geometry → not ok")
	}
	if _, _, ok := geoPoint(json.RawMessage(`null`)); ok {
		t.Fatal("null geometry → not ok")
	}
	// A hostile deeply-nested blob must not crash (P4 F2: the old any-decode
	// recursed per level and overflowed the stack). Token streaming is iterative.
	deep := json.RawMessage(strings.Repeat("[", 200000) + strings.Repeat("]", 200000))
	if _, _, ok := geoPoint(deep); ok {
		t.Fatal("a coordinate-less deeply-nested blob → not ok (and no crash)")
	}
}

func TestWithinFiltersByRadiusAndDropsPointless(t *testing.T) {
	// Center: San Diego. Near = ~20 km away; far = across the country.
	evs := []Event{
		{ID: "near", Lat: 32.90, Lon: -117.20, HasPoint: true},
		{ID: "far", Lat: 40.71, Lon: -74.01, HasPoint: true}, // New York
		{ID: "zone", HasPoint: false},                        // a zone-only NWS alert — no point
	}
	got := Within(evs, 32.72, -117.16, 50) // 50 mi around San Diego
	if len(got) != 1 || got[0].ID != "near" {
		t.Fatalf("only the near event within 50 mi, point-less dropped: %v", ids(got))
	}
	// A non-positive radius keeps everything (the global "All" ticker).
	if all := Within(evs, 32.72, -117.16, 0); len(all) != 3 {
		t.Fatalf("radius 0 = All (global): %v", ids(all))
	}
}

func TestLocateTiesToHighestWatchlistThenPlaceThenArea(t *testing.T) {
	watch := []snapshot.LocationRef{
		{Label: "San Diego, CA", Lat: 32.72, Lon: -117.16}, // highest
		{Label: "Oceanside, CA", Lat: 33.20, Lon: -117.38}, // also within range of the test point
	}
	// A quake ~20 km from San Diego (and within range of Oceanside too): the
	// HIGHEST watchlist location wins.
	if got := Locate(true, 32.90, -117.20, "10 km N of Chula Vista, CA", watch, nil); got != "San Diego, CA" {
		t.Fatalf("highest applicable watchlist location wins, got %q", got)
	}
	// Far from any watchlist location: the feed's named place, cleaned.
	if got := Locate(true, 27.9, 86.9, "55 km NW of Kodari, Nepal", watch, nil); got != "Kodari, Nepal" {
		t.Fatalf("clean place after ' of ', got %q", got)
	}
	// No clean place and no watch tie: the fuzzy metro area from nearest-city.
	nearest := func(lat, lon float64) string { return "Miami" }
	if got := Locate(true, 25.0, -70.0, "", watch, nearest); got != "the Miami area" {
		t.Fatalf("fuzzy area fallback, got %q", got)
	}
}

func TestLocateSkipsTheTieWithoutAPoint(t *testing.T) {
	watch := []snapshot.LocationRef{{Label: "Accra, GH", Lat: 0.0, Lon: 0.0}} // sits at Null Island
	// A zone-only NWS alert (no point, lat=lon=0) must NOT tie to a watchlist
	// location near (0,0), nor claim a nearest metro — it uses its place (P4 F8).
	nearest := func(lat, lon float64) string { return "Honolulu" }
	if got := Locate(false, 0, 0, "Oklahoma County, OK", watch, nearest); got != "Oklahoma County, OK" {
		t.Fatalf("no point → place only, got %q", got)
	}
	// With a real point at (0,0), the tie is allowed (the accident is intended).
	if got := Locate(true, 0, 0, "", watch, nearest); got != "Accra, GH" {
		t.Fatalf("a real point ties normally, got %q", got)
	}
}

func TestClampFieldBoundsHostileLength(t *testing.T) {
	if got := clampField("Tornado Warning"); got != "Tornado Warning" {
		t.Fatalf("a normal field is unchanged: %q", got)
	}
	huge := strings.Repeat("x", 5_000)
	if got := clampField(huge); len([]rune(got)) != maxFieldRunes {
		t.Fatalf("a hostile field is bounded to %d runes, got %d", maxFieldRunes, len([]rune(got)))
	}
}

func ids(evs []Event) []string {
	out := make([]string, len(evs))
	for i, e := range evs {
		out[i] = e.ID
	}
	return out
}
