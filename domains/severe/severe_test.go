package severe

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestNormalizeIDJoinsBothNWSForms(t *testing.T) {
	url := "https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.8b42a205bbf75387c6b5f8802e96a0992b4361aa.001.1"
	bare := "urn:oid:2.49.0.1.840.0.8b42a205bbf75387c6b5f8802e96a0992b4361aa.001.1"
	a, ok1 := NormalizeID(url)
	b, ok2 := NormalizeID(bare)
	if !ok1 || !ok2 || a != b || a != bare {
		t.Fatalf("forms differ: %q %q", a, b)
	}
	if k, ok := NormalizeID("https://evil.example/x/urn:oid:not-digits"); ok || k != "https://evil.example/x/urn:oid:not-digits" {
		t.Fatalf("a malformed OID must keep its raw id and never merge: %q %v", k, ok)
	}
	if k, ok := NormalizeID("https://evil.example/x/" + bare); ok || k == bare {
		t.Fatalf("a real OID under a foreign prefix must not merge: %q %v", k, ok)
	}
	if k, ok := NormalizeID("us7000tbwb"); ok || k != "us7000tbwb" {
		t.Fatalf("a USGS id passes through: %q %v", k, ok)
	}
}

func TestClassifySixTabs(t *testing.T) {
	cases := map[string]Tab{"Tornado Warning": TabWarnings, "Flash Flood Warning": TabWarnings, "Winter Storm Watch": TabWatches, "Flood Watch": TabWatches, "Heat Advisory": TabAdvisories, "Special Weather Statement": TabStatements}
	for product, want := range cases {
		if got, ok := Classify(globalfeed.ClassSevereWx, product); !ok || got != want {
			t.Errorf("%s → %v %v", product, got, ok)
		}
	}
	if got, ok := Classify(globalfeed.ClassSevereWx, "Air Quality Alert"); ok || got != TabNone {
		t.Error("Air Quality Alert must not be shown in v1, and must never index a tab")
	}
	if got, _ := Classify(globalfeed.ClassQuake, "Earthquake"); got != TabQuakes {
		t.Error("quake tab")
	}
	if got, _ := Classify(globalfeed.ClassTropical, "Hurricane"); got != TabTropical {
		t.Error("tropical tab")
	}
}

func TestUnionDedupsAcrossPathsAndPrefersTheLocationRecord(t *testing.T) {
	sent := time.Date(2026, 8, 28, 8, 45, 0, 0, time.UTC)
	feed := []globalfeed.Event{{
		ID: "https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.1.001.1", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
		Severity: globalfeed.SevRed, Location: "Olathe, KS", Lat: 38.9, Lon: -94.8, HasPoint: true, At: sent, Source: "NWS",
		Severe: &globalfeed.SevereDetail{Headline: "feed headline", SenderName: "NWS Kansas City", Sent: sent, MaxWindGust: "70 mph"},
	}}
	locs := []snapshot.Location{{Label: "Olathe", Lat: 38.9, Lon: -94.8, TZ: "America/Chicago", Alerts: []snapshot.Alert{{
		ID: "urn:oid:2.49.0.1.840.0.1.001.1", Event: "Tornado Warning", Severity: "extreme", Sent: sent, Onset: &sent, Expires: sent.Add(time.Hour),
		Headline: "location headline", Description: "desc", Instruction: "instr", SenderName: "NWS Kansas City",
	}}}}
	rows := Union(feed, locs, sent.Add(time.Minute))
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.Detail.Alert == nil || r.Detail.Alert.Headline != "location headline" {
		t.Fatalf("the location record must win: %+v", r)
	}
	if r.Detail.Severe == nil || r.Detail.Severe.MaxWindGust != "70 mph" {
		t.Fatalf("the national CAP parameters must be kept beside the location record: %+v", r.Detail)
	}
	if !r.HasPoint || r.Tied == nil || r.Tied.Label != "Olathe" {
		t.Fatalf("feed lat/lon and the tie must merge in: %+v", r)
	}
	if r.Severity != globalfeed.SevRed {
		t.Fatalf("the curated tier is authoritative: %v", r.Severity)
	}
}

func TestUnionGuardsSupersededOnTheLocationPath(t *testing.T) {
	older := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	newer := older.Add(30 * time.Minute)
	exp := newer.Add(2 * time.Hour)
	locs := []snapshot.Location{{Label: "L", Alerts: []snapshot.Alert{
		{ID: "urn:oid:1.1", Event: "Flood Warning", SenderName: "NWS A", Sent: older, Onset: &older, Expires: exp},
		{ID: "urn:oid:1.2", Event: "Flood Warning", SenderName: "NWS A", Sent: newer, Onset: &newer, Expires: exp, References: []string{"urn:oid:1.1"}},
		{ID: "urn:oid:1.3", Event: "Air Quality Alert", SenderName: "NWS B", Sent: newer, Onset: &newer, Expires: exp, References: []string{"urn:oid:1.2"}},
	}}}
	rows := Union(nil, locs, newer)
	keys := map[string]bool{}
	for _, r := range rows {
		keys[r.Key] = true
	}
	if keys["urn:oid:1.1"] || !keys["urn:oid:1.2"] {
		t.Fatalf("1.1 superseded by 1.2; 1.2 must NOT be hidden by the rogue 1.3: %v", keys)
	}
}

func TestUnionHonoursTheFeedsSupersededFlagOnTheLocationPath(t *testing.T) {
	now := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	feed := []globalfeed.Event{{ID: "https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.9.001.1", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Superseded: true, At: now}}
	locs := []snapshot.Location{{Label: "L", Alerts: []snapshot.Alert{{ID: "urn:oid:2.49.0.1.840.0.9.001.1", Event: "Tornado Warning", Sent: now, Onset: &now, Expires: now.Add(time.Hour)}}}}
	if rows := Union(feed, locs, now); len(rows) != 0 {
		t.Fatalf("a superseded alert resurfaced through the location path: %+v", rows)
	}
}

func TestUnionOneRowPerMultiLocationAlertAndDropsExpired(t *testing.T) {
	now := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	a := snapshot.Alert{ID: "urn:oid:2.49.0.1.840.0.5.001.1", Event: "Heat Advisory", Sent: now, Onset: &now, Expires: now.Add(time.Hour)}
	expired := snapshot.Alert{ID: "urn:oid:2.49.0.1.840.0.6.001.1", Event: "Wind Advisory", Sent: now, Onset: &now, Expires: now.Add(-time.Minute)}
	locs := []snapshot.Location{{Label: "First", Alerts: []snapshot.Alert{a, expired}}, {Label: "Second", Alerts: []snapshot.Alert{a}}}
	rows := Union(nil, locs, now)
	if len(rows) != 1 || rows[0].Tied.Label != "First" || rows[0].Tab != TabAdvisories {
		t.Fatalf("one row, tied to the first (highest) location, in Advisories: %+v", rows)
	}
}

func TestSortAndCap(t *testing.T) {
	var rows []Row
	base := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 600; i++ {
		rows = append(rows, Row{Key: "k" + string(rune('a'+i%26)) + string(rune('0'+i/26)), At: base.Add(time.Duration(i) * time.Minute)})
	}
	Sort(rows)
	if !rows[0].At.After(rows[1].At) {
		t.Fatal("not Declared DESC")
	}
	kept, total := Cap(rows, MaxRows)
	if total != 600 || len(kept) != MaxRows || !kept[0].At.Equal(base.Add(599*time.Minute)) {
		t.Fatalf("cap: kept %d of %d, newest %v", len(kept), total, kept[0].At)
	}
	byTab := ByTab(rows)
	if len(byTab[TabWarnings]) != 600 {
		t.Fatalf("ByTab: %d", len(byTab[TabWarnings]))
	}
}

// One product reads one tier by every path: a Tornado Watch is yellow on the
// curated national feed, and CAP "Severe" on the location path must not lift
// it to orange (R3-A-05).
func TestSeverityIsTheSameByEveryPath(t *testing.T) {
	now := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	feed := []globalfeed.Event{{ID: "https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.1.001.1", Class: globalfeed.ClassSevereWx, Type: "Tornado Watch",
		Severity: globalfeed.SevYellow, At: now, Until: now.Add(time.Hour), Severe: &globalfeed.SevereDetail{Sent: now}}}
	locs := []snapshot.Location{{Label: "L", Alerts: []snapshot.Alert{{ID: "urn:oid:2.49.0.1.840.0.1.001.1", Event: "Tornado Watch", Severity: "severe", Sent: now, Effective: now, Expires: now.Add(time.Hour)}}}}
	untied, locOnly, both := Union(feed, nil, now), Union(nil, locs, now), Union(feed, locs, now)
	if untied[0].Severity != globalfeed.SevYellow || locOnly[0].Severity != globalfeed.SevYellow || both[0].Severity != globalfeed.SevYellow {
		t.Fatalf("Tornado Watch: national=%v location=%v both=%v — want yellow everywhere", untied[0].Severity, locOnly[0].Severity, both[0].Severity)
	}
	if got := severityOf("Coastal Flood Advisory", "severe"); got != globalfeed.SevOrange {
		t.Fatalf("an uncurated product still reads its CAP severity: %v", got)
	}
}

// The Guard set applies to the feed path too: an alert a tracked location's
// NEWER update replaced must not resurface untied through the national feed
// that still lists the old id (R3-A-04). References arrive in the API's URL
// form on the location path.
func TestFeedPathHonoursTheLocationGuard(t *testing.T) {
	older := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	newer := older.Add(10 * time.Minute)
	exp := newer.Add(time.Hour)
	feed := []globalfeed.Event{{ID: "https://api.weather.gov/alerts/urn:oid:1.1", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", At: older, Until: exp,
		Severe: &globalfeed.SevereDetail{SenderName: "NWS A", Sent: older}}}
	locs := []snapshot.Location{{Label: "L", Alerts: []snapshot.Alert{
		{ID: "urn:oid:1.1", Event: "Tornado Warning", SenderName: "NWS A", Sent: older, Effective: older, Expires: exp},
		{ID: "urn:oid:1.2", Event: "Tornado Warning", SenderName: "NWS A", Sent: newer, Effective: newer, Expires: exp, References: []string{"https://api.weather.gov/alerts/urn:oid:1.1"}},
	}}}
	rows := Union(feed, locs, newer)
	if len(rows) != 1 || rows[0].Key != "urn:oid:1.2" || rows[0].Tied == nil {
		t.Fatalf("want only the tied replacement, got %+v", rows)
	}
}

// "Declared" is the ISSUE time; a hazard that begins later reads "Starts" in
// the record (HUM LEAD UAT 2026-08-28: an advisory issued at 09:00 for
// 20:00 showed a future "declared" and read as bad data).
func TestDeclaredIsTheIssueTimeAndTheOnsetIsStarts(t *testing.T) {
	sent := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	onset := sent.Add(11 * time.Hour)
	exp := onset.Add(6 * time.Hour)
	locs := []snapshot.Location{{Label: "Los Angeles", Alerts: []snapshot.Alert{{ID: "urn:oid:1.9", Event: "Heat Advisory", Severity: "moderate", Sent: sent, Effective: sent, Onset: &onset, Expires: exp}}}}
	rows := Union(nil, locs, sent.Add(time.Hour))
	if len(rows) != 1 || !rows[0].At.Equal(sent) {
		t.Fatalf("declared must be the issue time: %+v", rows)
	}
	rec := RecordOf(rows[0], time.UTC)
	if !strings.Contains(rec.Timing, "Declared 08/28 09:00 UTC") || !strings.Contains(rec.Timing, "Starts 08/28 20:00 UTC") || !strings.Contains(rec.Timing, "(~6h)") {
		t.Fatalf("timing: %q", rec.Timing)
	}
}

// An OID whose URL form is long still meets its bare form (REVIEW R5-B-05:
// ids were clamped to 120 runes before normalising, so the two paths
// disagreed above ~90-rune OIDs); and cap1 reads runes (R5-B-06).
func TestLongIDsMeetAcrossPathsAndCap1ReadsRunes(t *testing.T) {
	bare := "urn:oid:2.49.0.1.840.0." + strings.Repeat("1", 100)
	if key, ok := NormalizeID("https://api.weather.gov/alerts/" + bare); !ok || key != bare {
		t.Fatalf("the URL form (%d runes) normalises to the bare OID: %q %v", len([]rune("https://api.weather.gov/alerts/"+bare)), key, ok)
	}
	if got := cap1("élevé"); got != "Élevé" {
		t.Fatalf("cap1 by rune: %q", got)
	}
}

// The index holds no network client and never fetches (NFR-1): the
// package imports neither net/http nor the app's client — the counter the
// objectives asked for is this boundary (REVIEW R5-A-01).
func TestSevereImportsNoNetwork(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			if path := strings.Trim(imp.Path.Value, `"`); path == "net/http" || strings.Contains(path, "/httpx") || strings.Contains(path, "/domains/weather") || strings.Contains(path, "/domains/globalfeed/") {
				t.Fatalf("%s imports %s: the index must stay a pure join", name, path)
			}
		}
	}
}
