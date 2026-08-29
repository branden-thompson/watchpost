# P1 — Domain: widened feeds, bounded location alerts, `domains/severe`

Every task: RED test first, then the code, then verify. Import lint: `domains/*` may import `platform/*`
and sibling domain packages; nothing here imports `modes/*` or `app/*`.

---

## Task 1.1 — Detail structs and bounds

**File:** `domains/globalfeed/detail.go` (CREATE)

**Test first (RED):** `domains/globalfeed/detail_test.go`
```go
package globalfeed

import (
	"strings"
	"testing"
)

func TestClampProseBoundsHostileText(t *testing.T) {
	long := strings.Repeat("x", maxProseRunes+50)
	if got := clampProse(long); len([]rune(got)) != maxProseRunes {
		t.Fatalf("clampProse kept %d runes, want %d", len([]rune(got)), maxProseRunes)
	}
	if got := clampProse("short"); got != "short" {
		t.Fatalf("clampProse changed a short string: %q", got)
	}
}

func TestClampIntAndSlice(t *testing.T) {
	if clampInt(999, 0, 250) != 250 || clampInt(-5, 0, 250) != 0 || clampInt(45, 0, 250) != 45 {
		t.Fatal("clampInt bounds wrong")
	}
	s := make([]string, maxListLen+10)
	if got := clampSlice(s); len(got) != maxListLen {
		t.Fatalf("clampSlice kept %d, want %d", len(got), maxListLen)
	}
}
```

**Code:**
```go
package globalfeed

// detail.go — per-class detail carried on an Event (0.13.0, SAM-D-21): what
// the severe-events window renders beyond the tape's thin fields. Every string
// from a feed is bounded here (P4 F5 / NFR-5): short fields to maxFieldRunes,
// prose to maxProseRunes, numbers to their physical ranges, lists to maxListLen.

import "time"

// maxProseRunes bounds an NWS description/instruction: real products run 1–3 KB;
// the cap only stops a hostile feed from growing the window without bound.
const maxProseRunes = 4000

// maxListLen bounds a feed-supplied list (zones, references, parameters).
const maxListLen = 50

// QuakeDetail is a USGS significant-feed event's record (the render list in
// 02-analysis/data-shape.md §4 plus the Keep fields).
type QuakeDetail struct {
	Mag       *float64  // nil when the feed omits it (absent ≠ 0.0)
	MagType   string    // "mww"
	DepthKm   float64   // geometry[2]
	Title     string    // "M 5.8 - 55 km NW of Kodāri, Nepal"
	Alert     string    // PAGER: green | yellow | orange | red | ""
	Sig       int       // significance
	Felt      int       // DYFI responses
	CDI, MMI  *float64  // community / modelled intensity, nil when absent
	Status    string    // "reviewed" | "automatic"
	Tsunami   bool
	UpdatedAt time.Time // properties.updated
	URL       string    // Keep, never rendered as a link (S6)
	DetailURL string    // Keep, never fetched or rendered as a link
}

// TropicalDetail is an NHC current-storm record.
type TropicalDetail struct {
	Name         string // "Dolly"
	Basin        string // "the Atlantic"
	BinNumber    string // "AT4"
	WindKt       int
	PressureMb   int
	MoveDirDeg   int
	MoveSpeedKt  int
	LatText      string // "15.0N"
	LonText      string // "46.9W"
	AdvisoryNum  string // publicAdvisory.advNum
	AdvisoryAt   time.Time
	ForecastNum  string // forecastAdvisory.advNum
	DiscussionNum string
	AdvisoryURL  string // Keep, never rendered as a link
}

// SevereDetail is a CAP alert record from the national NWS query (the
// location path carries the same fields on snapshot.Alert).
type SevereDetail struct {
	Headline    string
	Description string // prose, ≤ maxProseRunes
	Instruction string // prose, ≤ maxProseRunes
	Severity    string // CAP: Extreme | Severe | Moderate | Minor | Unknown
	Certainty   string
	Urgency     string
	MessageType string // Alert | Update | Cancel
	Category    string
	Response    string
	SenderName  string
	Sender      string
	Effective   time.Time
	Sent        time.Time
	Expires     time.Time
	Ends        time.Time // zero when absent
	Onset       time.Time // zero when absent
	AffectedZones []string
	References  []string
	MaxWindGust string // parameters allowlist (S7)
	MaxHailSize string
	EventMotion string
	NWSHeadline string
	VTEC        string
}

// clampProse bounds prose to maxProseRunes (rune-safe).
func clampProse(s string) string {
	r := []rune(s)
	if len(r) <= maxProseRunes {
		return s
	}
	return string(r[:maxProseRunes])
}

// clampInt bounds n to [lo, hi].
func clampInt(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}

// clampSlice bounds a list to maxListLen entries, each to maxFieldRunes.
func clampSlice(s []string) []string {
	if len(s) > maxListLen {
		s = s[:maxListLen]
	}
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = clampField(v)
	}
	return out
}

// clampFloat bounds a physical quantity; NaN/Inf read as lo.
func clampFloat(f, lo, hi float64) float64 {
	if f != f || f < lo { // NaN compares false to itself
		return lo
	}
	if f > hi {
		return hi
	}
	return f
}
```

**Verify:** `go test ./domains/globalfeed -run 'TestClamp' -v`

---

## Task 1.2 — `Event.Name`, per-class detail pointers, name-aware `Sentence()`

**File:** `domains/globalfeed/event.go` (MODIFY)

**Test first (RED):** append to `domains/globalfeed/logic_test.go`
```go
func TestSentenceNamesAStormWithoutAnArticle(t *testing.T) {
	e := Event{Class: ClassTropical, Type: "Tropical Storm", Name: "Dolly", Location: "the Atlantic"}
	if got := e.Sentence(); got != "Tropical Storm Dolly has been reported for the Atlantic" {
		t.Fatalf("Sentence() = %q", got)
	}
	q := Event{Class: ClassQuake, Type: "Earthquake", Location: "Kodāri, Nepal"}
	if got := q.Sentence(); got != "An Earthquake has been recorded for Kodāri, Nepal" {
		t.Fatalf("unnamed Sentence() changed: %q", got)
	}
	if got := e.Title(); got != "Tropical Storm Dolly" {
		t.Fatalf("Title() = %q", got)
	}
}
```

**Code:** replace the `Event` struct and `Sentence()` in `event.go`; keep `Verb()`, `Article()`,
`clampField`, `maxFieldRunes` as they are.
```go
// Event is one global hazard event, unified across the three feeds.
type Event struct {
	ID         string    // stable source id — the dedup key (USGS id, NHC id, NWS alert id)
	Class      Class     // hazard family
	Severity   Severity  // the bg tier
	Type       string    // the spoken "<severe alert type>": "Earthquake", "Hurricane", "Tornado Warning"
	Name       string    // a named storm ("Dolly"); "" otherwise (0.13.0, SAM-D-14/20)
	Place      string    // the feed's raw place text, before D5 tying (locate.go resolves the spoken Location)
	Location   string    // the tied representative location (D5) — set by Locate
	Lat, Lon   float64   // the event point, for the watchlist tie and the nearest-city fuzzy area
	HasPoint   bool      // whether Lat/Lon is a real location (a zone-only NWS alert has none) — the radius filter excludes point-less alerts
	Superseded bool      // this alert was updated/replaced by a newer one (NWS references) — kept only so the ticker can seen-mark it, never displayed/announced
	At         time.Time // event / issue time — stack recency, the "declared/recorded" time
	Until      time.Time // active-window end (NWS ends/expires); zero = no expiry — a quake's instant, a storm the feed still lists (Active keeps it until the feed drops it)
	Source     string    // "USGS" | "NHC" | "NWS"

	// Per-class detail (0.13.0, SAM-D-21): exactly one is non-nil for a
	// parsed event; all nil on a thin event (tests, seen-store replay).
	Quake    *QuakeDetail
	Tropical *TropicalDetail
	Severe   *SevereDetail
}

// Title is the event as a headline: the type, plus the storm's name when it
// has one ("Tropical Storm Dolly", "Tornado Warning").
func (e Event) Title() string {
	if e.Name != "" {
		return e.Type + " " + e.Name
	}
	return e.Type
}

// Sentence is the event's lead line, shared by the marquee and the spoken
// narration (single owner): "A(n) <Type> has been <verb> for <Location>" — or,
// for a named storm, "<Type> <Name> has been <verb> for <Location>" with no
// article (HUM LEAD SAM-D-20; red-team C-3: folding the name into Type would
// make Article() say "A Tropical Storm Dolly").
func (e Event) Sentence() string {
	if e.Name != "" {
		return e.Title() + " has been " + e.Verb() + " for " + e.Location
	}
	return e.Article() + " " + e.Type + " has been " + e.Verb() + " for " + e.Location
}
```
Also update `app/ticker.go` `tapeText` (P2-4) to use `e.Title()` so the tape reads "Tropical Storm Dolly ·
the Atlantic …".

**Verify:** `go test ./domains/globalfeed -run 'TestSentence' -v`

---

> **Order note (red-team PLAN-CQ #11):** Tasks 1.3–1.5 call the memo type Task 1.7 defines. Execute
> **1.7 first** (it is self-contained: `memo.go` + its test), then 1.3 → 1.5; `cov_test.go` (1.10) uses
> 1.3's `fetchFixture`. The batch's verify gates are stated per task; the compile gate is "after 1.7 + 1.3".

## Task 1.3 — USGS parser: the render list

**File:** `domains/globalfeed/usgs.go` (MODIFY)

**Test first (RED):** `domains/globalfeed/detail_test.go` (append)
```go
func TestUSGSDecodesTheRenderList(t *testing.T) {
	evs := fetchFixture(t, "usgs_significant_week.json", func(c *httpx.Client, base string) Source { return NewUSGS(c, base) })
	if len(evs) == 0 {
		t.Fatal("no events from the fixture")
	}
	q := evs[0].Quake
	if q == nil {
		t.Fatal("Quake detail nil")
	}
	// COV: every field in the render list is populated from the fixture (data-shape.md §4).
	if q.Mag == nil || q.MagType == "" || q.DepthKm == 0 || q.Title == "" || q.Status == "" || q.UpdatedAt.IsZero() || q.Sig == 0 {
		t.Fatalf("render list incomplete: %+v", *q)
	}
	if evs[0].Name != "" {
		t.Fatal("a quake has no name")
	}
}

// fetchFixture serves a testdata file on a local server and runs one Source.Fetch.
func fetchFixture(t *testing.T, name string, mk func(*httpx.Client, string) Source) []Event {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	c, err := httpx.New(httpx.Config{UserAgent: "test"})
	if err != nil {
		t.Fatal(err)
	}
	evs, err := mk(c, srv.URL).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return evs
}
```
(imports: `context`, `net/http`, `net/http/httptest`, `os`, `path/filepath`,
`github.com/branden-thompson/watchpost/platform/httpx`.) The fixture is Task 1.10's copy of
`domains/globalfeed/testdata/usgs_significant_week.json` (the probe sample, already committed there at PLAN exit).

**Code:** replace `usgsFeed` and the loop body in `Fetch`.
```go
// usgsFeed is the slice of the summary GeoJSON the ticker and the severe
// window read (the render list of data-shape.md §4; the network/telemetry tail
// is not decoded in v1 — SAM-D-24 E-2).
type usgsFeed struct {
	Features []struct {
		ID         string `json:"id"`
		Properties struct {
			Mag     *float64 `json:"mag"`
			MagType string   `json:"magType"`
			Place   string   `json:"place"`
			Time    int64    `json:"time"`    // epoch ms
			Updated int64    `json:"updated"` // epoch ms
			Type    string   `json:"type"`
			Title   string   `json:"title"`
			Alert   string   `json:"alert"`
			Status  string   `json:"status"`
			Tsunami int      `json:"tsunami"`
			Sig     int      `json:"sig"`
			Felt    *int     `json:"felt"`
			CDI     *float64 `json:"cdi"`
			MMI     *float64 `json:"mmi"`
			URL     string   `json:"url"`
			Detail  string   `json:"detail"`
		} `json:"properties"`
		Geometry struct {
			Coordinates []float64 `json:"coordinates"` // [lon, lat, depth]
		} `json:"geometry"`
	} `json:"features"`
}

// Fetch reads the feed through the parse memo (Task 1.7): the body comes from
// httpx (cache/TTL/conditional GET as before); it is decoded only when changed.
func (u *USGS) Fetch(ctx context.Context) ([]Event, error) {
	return u.memo.events(func() ([]byte, bool, error) {
		var body []byte
		hdr, err := u.client.GetJSON(ctx, u.base, &body, httpx.TTL(usgsTTL)) // read-only slice (the GetText contract)
		return body, hdr == nil, err
	}, u.parse)
}

// parse decodes one body into events (pure; the memo calls it on change).
func (u *USGS) parse(body []byte) ([]Event, error) {
	var feed usgsFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		u.client.Forget(u.base) // the GetJSON poison guard, kept on the raw-body path
		return nil, fmt.Errorf("USGS: bad response body: %w", err)
	}
	out := make([]Event, 0, len(feed.Features))
	for _, f := range feed.Features {
		g := f.Geometry.Coordinates
		if f.ID == "" || f.Properties.Mag == nil || len(g) < 2 {
			continue
		}
		mag := clampFloat(*f.Properties.Mag, -1, 12)
		at := time.UnixMilli(f.Properties.Time).UTC()
		if f.Properties.Time == 0 {
			at = time.Now() // a missing epoch is "as of now", not 1970 (P4 F6)
		}
		p := f.Properties
		d := &QuakeDetail{
			Mag: &mag, MagType: clampField(p.MagType), Title: clampField(p.Title),
			Alert: pagerAlert(p.Alert), Sig: clampInt(p.Sig, 0, 5000), Status: clampField(p.Status),
			Tsunami: p.Tsunami != 0, URL: clampField(p.URL), DetailURL: clampField(p.Detail),
		}
		if len(g) >= 3 {
			d.DepthKm = clampFloat(g[2], 0, 1000)
		}
		if p.Felt != nil {
			d.Felt = clampInt(*p.Felt, 0, 1_000_000)
		}
		if p.CDI != nil {
			v := clampFloat(*p.CDI, 0, 12)
			d.CDI = &v
		}
		if p.MMI != nil {
			v := clampFloat(*p.MMI, 0, 12)
			d.MMI = &v
		}
		if p.Updated > 0 {
			d.UpdatedAt = time.UnixMilli(p.Updated).UTC()
		}
		out = append(out, Event{
			ID:       f.ID,
			Class:    ClassQuake,
			Severity: quakeSeverity(mag, p.Tsunami != 0),
			Type:     clampField(quakeType(p.Type)), // bounded feed field (P4 F5)
			Place:    clampField(p.Place),
			Lat:      g[1],
			Lon:      g[0],
			HasPoint: true,
			At:       at,
			Source:   u.Name(),
			Quake:    d,
		})
	}
	return out, nil
}

// pagerAlert validates the PAGER tier against its enum; anything else is "".
func pagerAlert(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "green", "yellow", "orange", "red":
		return strings.ToLower(strings.TrimSpace(s))
	}
	return ""
}
```

Add `memo sourceMemo` to the `USGS` struct and `"encoding/json"`, `"fmt"` to the imports (the memo type is
Task 1.7 — executed first, see the order note above).

**Verify:** `go test ./domains/globalfeed -run 'TestUSGS' -v`

---

## Task 1.4 — NHC parser: name, intensity, pressure, movement, advisories

**File:** `domains/globalfeed/nhc.go` (MODIFY)

**Test first (RED):** `domains/globalfeed/detail_test.go` (append)
```go
func TestNHCKeepsTheStormName(t *testing.T) {
	evs := fetchFixture(t, "nhc_currentstorms.json", func(c *httpx.Client, base string) Source { return NewNHC(c, base) })
	if len(evs) == 0 {
		t.Fatal("no storms from the fixture")
	}
	e := evs[0]
	if e.Name == "" || e.Tropical == nil || e.Tropical.Name != e.Name {
		t.Fatalf("name not carried: %+v", e)
	}
	if e.Tropical.WindKt == 0 || e.Tropical.PressureMb == 0 || e.Tropical.LatText == "" {
		t.Fatalf("render list incomplete: %+v", *e.Tropical)
	}
	if !strings.Contains(e.Sentence(), e.Name) {
		t.Fatalf("Sentence() does not name the storm: %q", e.Sentence())
	}
}
```

**Code:** replace `nhcFeed` and `Fetch`.
```go
type nhcAdvisory struct {
	AdvNum   string `json:"advNum"`
	Issuance string `json:"issuance"`
	URL      string `json:"url"`
}

type nhcFeed struct {
	ActiveStorms []struct {
		ID             string      `json:"id"`
		Name           string      `json:"name"`
		Classification string      `json:"classification"`
		BinNumber      string      `json:"binNumber"`
		Intensity      string      `json:"intensity"` // knots, as a string in the feed
		Pressure       string      `json:"pressure"`  // mb, as a string in the feed
		Latitude       string      `json:"latitude"`
		Longitude      string      `json:"longitude"`
		LatitudeNum    float64     `json:"latitudeNumeric"`
		LongitudeNum   float64     `json:"longitudeNumeric"`
		MovementDir    int         `json:"movementDir"`
		MovementSpeed  int         `json:"movementSpeed"`
		LastUpdate     string      `json:"lastUpdate"`
		PublicAdvisory nhcAdvisory `json:"publicAdvisory"`
		ForecastAdv    nhcAdvisory `json:"forecastAdvisory"`
		Discussion     nhcAdvisory `json:"forecastDiscussion"`
	} `json:"activeStorms"`
}

func (n *NHC) Fetch(ctx context.Context) ([]Event, error) {
	return n.memo.events(func() ([]byte, bool, error) {
		var body []byte
		hdr, err := n.client.GetJSON(ctx, n.base, &body, httpx.TTL(nhcTTL))
		return body, hdr == nil, err
	}, n.parse)
}

func (n *NHC) parse(body []byte) ([]Event, error) {
	var feed nhcFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		n.client.Forget(n.base)
		return nil, fmt.Errorf("NHC: bad response body: %w", err)
	}
	out := make([]Event, 0, len(feed.ActiveStorms))
	for _, s := range feed.ActiveStorms {
		typ, sev, ok := tropicalClass(s.Classification)
		if !ok || s.ID == "" {
			continue // not an active cyclone class (a low/disturbance/post-tropical) — skip
		}
		at, _ := time.Parse(time.RFC3339, s.LastUpdate)
		if at.IsZero() {
			at = time.Now() // a malformed advisory time is "as of now", not year 1 (P4 F6)
		}
		advAt, _ := time.Parse(time.RFC3339, s.PublicAdvisory.Issuance)
		name := clampField(strings.TrimSpace(s.Name))
		out = append(out, Event{
			ID:       s.ID,
			Class:    ClassTropical,
			Severity: sev,
			Type:     typ,
			Name:     name, // SAM-D-14/20: the storm is announced by name
			Place:    tropicalBasin(s.ID),
			Lat:      s.LatitudeNum,
			Lon:      s.LongitudeNum,
			HasPoint: true,
			At:       at.UTC(),
			Source:   n.Name(),
			Tropical: &TropicalDetail{
				Name: name, Basin: tropicalBasin(s.ID), BinNumber: clampField(s.BinNumber),
				WindKt: clampInt(atoiLoose(s.Intensity), 0, 250), PressureMb: clampInt(atoiLoose(s.Pressure), 0, 1100),
				MoveDirDeg: clampInt(s.MovementDir, 0, 360), MoveSpeedKt: clampInt(s.MovementSpeed, 0, 100),
				LatText: clampField(s.Latitude), LonText: clampField(s.Longitude),
				AdvisoryNum: clampField(s.PublicAdvisory.AdvNum), AdvisoryAt: advAt.UTC(),
				ForecastNum: clampField(s.ForecastAdv.AdvNum), DiscussionNum: clampField(s.Discussion.AdvNum),
				AdvisoryURL: clampField(s.PublicAdvisory.URL),
			},
		})
	}
	return out, nil
}

// atoiLoose reads the feed's numeric strings ("45", "999"); anything else is 0.
func atoiLoose(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
```
Add `memo sourceMemo` to the `NHC` struct and `"encoding/json"`, `"fmt"`, `"strconv"` to the imports.

**Verify:** `go test ./domains/globalfeed -run 'TestNHC' -v`

---

## Task 1.5 — NWS national parser: CAP fields, sender, parameters allowlist, guarded supersede

**File:** `domains/globalfeed/nws.go` (MODIFY) + `domains/globalfeed/supersede.go` (CREATE)

**Test first (RED):** `domains/globalfeed/detail_test.go` (append)
```go
func TestSupersedesOnlySameSenderProductNewer(t *testing.T) {
	older := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	live := Ref{ID: "urn:oid:1", Sender: "NWS Wichita KS", Product: "Tornado Warning", Sent: older}
	update := Ref{ID: "urn:oid:2", Sender: "NWS Wichita KS", Product: "Tornado Warning", Sent: newer, Replaces: []string{"urn:oid:1"}}
	rogue := Ref{ID: "urn:oid:3", Sender: "NWS Elsewhere", Product: "Air Quality Alert", Sent: newer, Replaces: []string{"urn:oid:1"}}
	if got := SupersededBy([]Ref{live, update}); !got["urn:oid:1"] {
		t.Fatal("a same-sender, same-product, newer update must supersede")
	}
	if got := SupersededBy([]Ref{live, rogue}); got["urn:oid:1"] {
		t.Fatal("a different sender/product must NOT supersede a live warning (red-team S3)")
	}
	stale := Ref{ID: "urn:oid:4", Sender: "NWS Wichita KS", Product: "Tornado Warning", Sent: older.Add(-time.Hour), Replaces: []string{"urn:oid:1"}}
	if got := SupersededBy([]Ref{live, stale}); got["urn:oid:1"] {
		t.Fatal("an OLDER message must not supersede")
	}
}

func TestNWSDecodesTheRenderListAndAllowlistsParameters(t *testing.T) {
	evs := fetchFixture(t, "nws_active_unfiltered_trimmed.json", func(c *httpx.Client, base string) Source { return NewNWS(c, base) })
	if len(evs) == 0 {
		t.Fatal("no alerts from the fixture")
	}
	var sws *Event
	for i := range evs {
		if evs[i].Type == "Special Weather Statement" {
			sws = &evs[i]
		}
	}
	if sws == nil || sws.Severe == nil {
		t.Fatal("Special Weather Statement not decoded")
	}
	d := sws.Severe
	if d.Headline == "" || d.Description == "" || d.SenderName == "" || d.Severity == "" || d.Sent.IsZero() {
		t.Fatalf("render list incomplete: %+v", *d)
	}
	if d.MaxWindGust == "" && d.NWSHeadline == "" && d.VTEC == "" {
		t.Fatal("no allowlisted parameter decoded (the fixture carries maxWindGust/NWSheadline/VTEC)")
	}
}
```
(the test server ignores the query string, so the curated `event=` filter does not filter the fixture.)

**Code — `supersede.go`:**
```go
package globalfeed

// supersede.go — the guarded superseded rule (0.13.0, NFR-12; red-team S3):
// an alert is superseded only by a NEWER message from the SAME sender for the
// SAME product that references it. Before the guard, any feature's references
// marked any id, so a crafted low-grade alert could hide a live warning.
// Shared by the national feed (nws.go) and, through domains/severe, the
// tracked-location path.

import "time"

// Ref is what the rule needs to know about one CAP message.
type Ref struct {
	ID       string
	Sender   string
	Product  string
	Sent     time.Time
	Replaces []string // the ids this message updates/replaces (CAP references)
}

// Supersedes reports whether newer legitimately replaces older.
func Supersedes(newer, older Ref) bool {
	if newer.ID == "" || older.ID == "" || newer.ID == older.ID {
		return false
	}
	if newer.Sender != older.Sender || newer.Product != older.Product {
		return false
	}
	if !newer.Sent.After(older.Sent) {
		return false
	}
	for _, r := range newer.Replaces {
		if r == older.ID {
			return true
		}
	}
	return false
}

// SupersededBy returns the set of ids in refs that a sibling legitimately
// supersedes.
func SupersededBy(refs []Ref) map[string]bool {
	byID := make(map[string]Ref, len(refs))
	for _, r := range refs {
		if r.ID != "" {
			byID[r.ID] = r
		}
	}
	out := make(map[string]bool)
	for _, newer := range refs {
		for _, target := range newer.Replaces {
			if older, ok := byID[target]; ok && Supersedes(newer, older) {
				out[target] = true
			}
		}
	}
	return out
}
```

**Code — `nws.go`:** replace `nwsFeed` and `Fetch`.
```go
type nwsFeed struct {
	Features []struct {
		ID         string `json:"id"` // the alert URI — stable
		Properties struct {
			Event       string `json:"event"`
			AreaDesc    string `json:"areaDesc"`
			Headline    string `json:"headline"`
			Description string `json:"description"`
			Instruction string `json:"instruction"`
			Severity    string `json:"severity"`
			Certainty   string `json:"certainty"`
			Urgency     string `json:"urgency"`
			MessageType string `json:"messageType"`
			Category    string `json:"category"`
			Response    string `json:"response"`
			Sender      string `json:"sender"`
			SenderName  string `json:"senderName"`
			Onset       string `json:"onset"`
			Sent        string `json:"sent"`
			Effective   string `json:"effective"`
			Ends        string `json:"ends"`    // when the hazard ends (may be empty)
			Expires     string `json:"expires"` // when the alert message expires — always present
			AffectedZones []string `json:"affectedZones"`
			References  []struct {
				ID string `json:"@id"` // a prior alert this message updates/replaces
			} `json:"references"`
			Parameters map[string][]string `json:"parameters"` // read through the allowlist only (S7)
		} `json:"properties"`
		Geometry struct {
			Coordinates json.RawMessage `json:"coordinates"` // GeoJSON Point/Polygon/MultiPolygon, or absent (zone-only)
		} `json:"geometry"`
	} `json:"features"`
}

// parseCAPTime reads an RFC3339 field; zero when absent or malformed.
func parseCAPTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t.UTC()
}

// firstParam reads one allowlisted CAP parameter (the first value), bounded.
func firstParam(params map[string][]string, key string) string {
	if v, ok := params[key]; ok && len(v) > 0 {
		return clampField(v[0])
	}
	return ""
}

func (n *NWS) Fetch(ctx context.Context) ([]Event, error) {
	url := n.url()
	return n.memo.events(func() ([]byte, bool, error) {
		var body []byte
		hdr, err := n.client.GetJSON(ctx, url, &body, httpx.TTL(nwsTTL))
		return body, hdr == nil, err
	}, func(body []byte) ([]Event, error) { return n.parse(body, url) })
}

func (n *NWS) parse(body []byte, url string) ([]Event, error) {
	var feed nwsFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		n.client.Forget(url)
		return nil, fmt.Errorf("NWS: bad response body: %w", err)
	}
	// NWS issues a NEW id when it updates/replaces a warning, linking the prior
	// via `references`. The guarded rule (supersede.go) drops the prior only
	// when the update is a newer message from the same sender for the same
	// product — never on a bare reference (red-team S3).
	refs := make([]Ref, 0, len(feed.Features))
	for _, f := range feed.Features {
		r := Ref{ID: f.ID, Sender: f.Properties.SenderName, Product: f.Properties.Event, Sent: parseCAPTime(f.Properties.Sent)} // SenderName on BOTH paths (the location path has no `sender` email) — red-team PLAN S3
		for _, x := range f.Properties.References {
			if x.ID != "" && x.ID != f.ID {
				r.Replaces = append(r.Replaces, x.ID)
			}
		}
		refs = append(refs, r)
	}
	superseded := SupersededBy(refs)
	out := make([]Event, 0, len(feed.Features))
	for _, f := range feed.Features {
		p := f.Properties
		if f.ID == "" || p.Event == "" {
			continue
		}
		when := p.Onset
		if when == "" {
			when = p.Sent
		}
		at, _ := time.Parse(time.RFC3339, when)
		if at.IsZero() {
			at = time.Now() // a malformed/absent time is "as of now", not year 1 (P4 F6)
		}
		until := p.Ends
		if until == "" {
			until = p.Expires
		}
		ends, _ := time.Parse(time.RFC3339, until)
		lat, lon, ok := geoPoint(f.Geometry.Coordinates)
		d := &SevereDetail{
			Headline: clampField(p.Headline), Description: clampProse(p.Description), Instruction: clampProse(p.Instruction),
			Severity: clampField(p.Severity), Certainty: clampField(p.Certainty), Urgency: clampField(p.Urgency),
			MessageType: clampField(p.MessageType), Category: clampField(p.Category), Response: clampField(p.Response),
			SenderName: clampField(p.SenderName), Sender: clampField(p.Sender),
			Effective: parseCAPTime(p.Effective), Sent: parseCAPTime(p.Sent), Expires: parseCAPTime(p.Expires),
			Ends: parseCAPTime(p.Ends), Onset: parseCAPTime(p.Onset),
			AffectedZones: clampSlice(p.AffectedZones),
			MaxWindGust: firstParam(p.Parameters, "maxWindGust"), MaxHailSize: firstParam(p.Parameters, "maxHailSize"),
			EventMotion: firstParam(p.Parameters, "eventMotionDescription"), NWSHeadline: firstParam(p.Parameters, "NWSheadline"),
			VTEC: firstParam(p.Parameters, "VTEC"),
		}
		for _, x := range p.References {
			d.References = append(d.References, x.ID)
		}
		d.References = clampSlice(d.References)
		out = append(out, Event{
			ID:         f.ID,
			Class:      ClassSevereWx,
			Severity:   severeSeverity(p.Event),
			Type:       clampField(p.Event),    // the spoken type ("Tornado Warning"); bounded (P4 F5)
			Place:      clampField(p.AreaDesc), // the location; bounded likewise
			Lat:        lat,
			Lon:        lon,
			HasPoint:   ok,
			Superseded: superseded[f.ID],
			At:         at.UTC(),
			Until:      ends.UTC(),
			Source:     n.Name(),
			Severe:     d,
		})
	}
	return out, nil
}
```

Add `memo sourceMemo` to the `NWS` struct and `"fmt"` to the imports (`encoding/json` is already imported).
Test-file imports for `detail_test.go` (one block for Tasks 1.1–1.7):
```go
import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)
```

**Verify:** `go test ./domains/globalfeed -run 'TestSupersedes|TestNWS' -v`

---

## Task 1.6 — Location alerts: bounds + `SenderName`

**Files:** `platform/snapshot/types.go` (MODIFY), `domains/weather/nws/alerts.go` (MODIFY)

**Test first (RED):** `domains/weather/nws/alerts_test.go` (CREATE — the package has no alerts test file yet)
```go
package nws

import (
	"strconv"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestMapAlertBoundsEveryFieldAndKeepsSender(t *testing.T) {
	long := strings.Repeat("y", 10_000)
	pr := alertProps{ID: "urn:oid:1", Event: long, Headline: long, Description: long, Instruction: long, SenderName: "NWS Test", AreaDesc: long}
	for i := 0; i < 200; i++ {
		pr.AffectedZones = append(pr.AffectedZones, "https://api.weather.gov/zones/forecast/Z"+strconv.Itoa(i))
		pr.References = append(pr.References, struct {
			ID string `json:"@id"`
		}{ID: "urn:oid:r" + strconv.Itoa(i)})
	}
	perKey := map[snapshot.LocationKey][]snapshot.Alert{}
	mapAlert(pr, map[string][]snapshot.LocationKey{"Z1": {"k"}}, perKey)
	a := perKey["k"][0]
	if len([]rune(a.Event)) > maxFieldRunes || len([]rune(a.Description)) > maxProseRunes || len(a.AffectedZones) > maxListLen || len(a.References) > maxListLen {
		t.Fatalf("unbounded: event %d desc %d zones %d refs %d", len(a.Event), len(a.Description), len(a.AffectedZones), len(a.References))
	}
	if a.SenderName != "NWS Test" {
		t.Fatalf("SenderName lost: %q", a.SenderName)
	}
}
```

**Code — `platform/snapshot/types.go`:** add to `Alert` after `Instruction`:
```go
	SenderName    string     `json:"sender_name"` // the issuing office ("NWS Wichita KS") — the superseded guard's key (0.13.0, NFR-12)
```

**Code — `domains/weather/nws/alerts.go`:** add the bounds block and use it in `mapAlert`.
```go
// Field bounds for a CAP alert reaching the snapshot (0.13.0, NFR-5; red-team
// S2 — the location path was unbounded while the ticker path was not).
const (
	maxFieldRunes = 120
	maxProseRunes = 4000
	maxListLen    = 50
)

func clampRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func clampList(s []string) []string {
	if len(s) > maxListLen {
		s = s[:maxListLen]
	}
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = clampRunes(v, maxFieldRunes)
	}
	return out
}
```
Add `SenderName string \`json:"senderName"\`` to `alertProps`, and rewrite the `a := snapshot.Alert{…}`
literal and the zone/reference loops:
```go
	a := snapshot.Alert{
		ID:          clampRunes(pr.ID, maxFieldRunes),
		Event:       clampRunes(pr.Event, maxFieldRunes),
		Severity:    strings.ToLower(clampRunes(pr.Severity, maxFieldRunes)),
		Urgency:     strings.ToLower(clampRunes(pr.Urgency, maxFieldRunes)),
		Certainty:   strings.ToLower(clampRunes(pr.Certainty, maxFieldRunes)),
		MessageType: strings.ToLower(clampRunes(pr.MessageType, maxFieldRunes)),
		Sent:        pr.Sent, Effective: pr.Effective, Onset: pr.Onset,
		Expires: pr.Expires, Ends: pr.Ends,
		AreaDesc: clampRunes(pr.AreaDesc, maxFieldRunes), Headline: clampRunes(pr.Headline, maxFieldRunes),
		Description: clampRunes(pr.Description, maxProseRunes), Instruction: clampRunes(pr.Instruction, maxProseRunes),
		SenderName:  clampRunes(pr.SenderName, maxFieldRunes),
		Source: snapshot.SourceInfo{Provider: "nws", IssuedAt: pr.Sent},
	}
	var refs []string
	for _, r := range pr.References {
		refs = append(refs, r.ID)
	}
	a.References = clampList(refs)
	var zones []string
	for _, zURL := range pr.AffectedZones {
		zones = append(zones, lastSegment(zURL))
	}
	a.AffectedZones = clampList(zones)
```
(the matching loop below it is unchanged: it ranges over `a.AffectedZones`.)

**Verify:** `go test ./domains/weather/nws ./platform/snapshot -run 'TestMapAlert' -v`

---

## Task 1.7 — Parse memo: skip decode when the body is unchanged

**File:** `domains/globalfeed/memo.go` (CREATE); wire into the three `Fetch` methods.

**Test first (RED):** `domains/globalfeed/detail_test.go` (append)
```go
func TestSourceMemoSkipsDecodeOnAnUnchangedBody(t *testing.T) {
	decodes := 0
	m := &sourceMemo{}
	body := []byte(`{"a":1}`)
	get := func() ([]byte, bool, error) { return body, false, nil } // not a cache hit, same bytes
	parse := func([]byte) ([]Event, error) { decodes++; return []Event{{ID: "x"}}, nil }
	for i := 0; i < 3; i++ {
		evs, err := m.events(get, parse)
		if err != nil || len(evs) != 1 {
			t.Fatal(err, evs)
		}
	}
	if decodes != 1 {
		t.Fatalf("decoded %d times for one body, want 1", decodes)
	}
	body = []byte(`{"a":2}`)
	if _, _ = m.events(get, parse); decodes != 2 {
		t.Fatalf("a changed body must re-decode: %d", decodes)
	}
}
```

**Code:**
```go
package globalfeed

// memo.go — the parse memo (0.13.0, NFR-3; red-team P7/S8): a source's body is
// fetched through httpx every cycle (its TTL/conditional GET decide whether the
// network is touched), but it is DECODED only when the bytes changed. Keyed on
// httpx's own "served from cache" fact first (hdr == nil), else on a sha256 of
// the body — 1–2 ms on the 1 MB national pull, against a 2-minute cadence.

import (
	"crypto/sha256"
	"sync"
)

type sourceMemo struct {
	mu     sync.Mutex
	ok     bool
	hash   [32]byte
	events []Event
}

// events returns the parsed events for the current body, decoding only when
// the body differs from the memoised one. get returns the body and whether
// httpx served it from cache untouched; parse decodes it.
func (m *sourceMemo) events(get func() ([]byte, bool, error), parse func([]byte) ([]Event, error)) ([]Event, error) {
	body, cached, err := get()
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ok && cached {
		return cloneEvents(m.events), nil
	}
	h := sha256.Sum256(body)
	if m.ok && h == m.hash {
		return cloneEvents(m.events), nil
	}
	evs, err := parse(body)
	if err != nil {
		return nil, err // an error is never memoised — the next cycle retries (S8)
	}
	m.ok, m.hash, m.events = true, h, evs
	return cloneEvents(evs), nil
}

// cloneEvents hands callers their own slice (Locate writes into elements).
func cloneEvents(in []Event) []Event {
	out := make([]Event, len(in))
	copy(out, in)
	return out
}
```
Why not `fire.Memo[T]` (`domains/fire/memo.go:13-30`): that memo memoises errors until `Forget` (right for
a parsed archive); a feed parse error must NOT be memoised — the next cycle retries (red-team S8) — and the
caller needs its own slice (Locate writes into elements). Two semantics, two small types; noted so the
"repeated code" check reads the reason here.

The three sources are already wired (Tasks 1.3–1.5 define `parse` + the memo-calling `Fetch`). One
table-driven read-only guard covers all three parsers (the `GetText` contract):
```go
func TestParsersMustNotMutateTheCacheBody(t *testing.T) {
	cases := []struct {
		file  string
		parse func([]byte) ([]Event, error)
	}{
		{"usgs_significant_week.json", (&USGS{}).parse},
		{"nhc_currentstorms.json", (&NHC{}).parse},
		{"nws_active_unfiltered_trimmed.json", func(b []byte) ([]Event, error) { return (&NWS{}).parse(b, "x") }},
	}
	for _, c := range cases {
		body, err := os.ReadFile(filepath.Join("testdata", c.file))
		if err != nil {
			t.Fatal(err)
		}
		before := append([]byte(nil), body...)
		if _, err := c.parse(body); err != nil {
			t.Fatal(c.file, err)
		}
		if !bytes.Equal(before, body) {
			t.Fatalf("%s: parse mutated the cache's body", c.file)
		}
	}
}
```
(`(&USGS{}).parse` with a nil client: `Forget` is only reached on a decode error, which the fixtures do not
trigger; a decode-error test constructs the client.)

**Verify:** `go test ./domains/globalfeed -run 'TestSourceMemo|TestParsersMustNotMutate|TestUSGS|TestNHC|TestNWS' -v`

---

## Task 1.8 — `domains/severe`: the unified index

**File:** `domains/severe/severe.go` (CREATE)

**Test first (RED):** `domains/severe/severe_test.go`
```go
package severe

import (
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
	if k, ok := NormalizeID("us7000tbwb"); ok || k != "us7000tbwb" {
		t.Fatalf("a USGS id passes through: %q %v", k, ok)
	}
}

func TestClassifySixTabs(t *testing.T) {
	cases := map[string]Tab{"Tornado Warning": TabWarnings, "Winter Storm Watch": TabWatches, "Heat Advisory": TabAdvisories, "Special Weather Statement": TabStatements}
	for product, want := range cases {
		if got, ok := Classify(globalfeed.ClassSevereWx, product); !ok || got != want {
			t.Errorf("%s → %v %v", product, got, ok)
		}
	}
	if _, ok := Classify(globalfeed.ClassSevereWx, "Air Quality Alert"); ok {
		t.Error("Air Quality Alert must not be shown in v1")
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
		Severe: &globalfeed.SevereDetail{Headline: "feed headline", Sender: "w-ict@noaa.gov", Sent: sent},
	}}
	locs := []snapshot.Location{{Label: "Olathe", Lat: 38.9, Lon: -94.8, TZ: "America/Chicago", Alerts: []snapshot.Alert{{
		ID: "urn:oid:2.49.0.1.840.0.1.001.1", Event: "Tornado Warning", Severity: "extreme", Sent: sent, Onset: &sent,
		Headline: "location headline", Description: "desc", Instruction: "instr", SenderName: "NWS Wichita KS",
	}}}}
	rows := Union(feed, locs, sent.Add(time.Minute))
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.Detail.Alert == nil || r.Detail.Alert.Headline != "location headline" {
		t.Fatalf("the location record must win: %+v", r)
	}
	if !r.HasPoint || r.Tied == nil || r.Tied.Label != "Olathe" {
		t.Fatalf("feed lat/lon and the tie must merge in: %+v", r)
	}
}

func TestUnionGuardsSupersededOnTheLocationPath(t *testing.T) {
	older := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	newer := older.Add(30 * time.Minute)
	locs := []snapshot.Location{{Label: "L", Alerts: []snapshot.Alert{
		{ID: "urn:oid:1.1", Event: "Flood Warning", SenderName: "NWS A", Sent: older, Onset: &older},
		{ID: "urn:oid:1.2", Event: "Flood Warning", SenderName: "NWS A", Sent: newer, Onset: &newer, References: []string{"urn:oid:1.1"}},
		{ID: "urn:oid:1.3", Event: "Air Quality Alert", SenderName: "NWS B", Sent: newer, Onset: &newer, References: []string{"urn:oid:1.2"}},
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
}
```

**Code:**
```go
// Package severe is the unified severe-event index (0.13.0, SAM-D-26 AX-1 = A):
// the global feeds' events (domains/globalfeed) and the tracked locations'
// alerts (platform/snapshot) folded into one de-duplicated, classified,
// sorted, capped list of rows, plus the [A]-shaped record of any row. Pure —
// no goroutines, no UI types — so the TTY window, report mode and any future
// surface consume the same index.
package severe

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Tab is the window's category, in importance order (HUM LEAD SAM-D-10).
type Tab int

const (
	TabWarnings Tab = iota
	TabWatches
	TabAdvisories
	TabStatements
	TabQuakes
	TabTropical
	NumTabs
	TabNone Tab = -1 // Classify's "not shown" — never a valid index (red-team PLAN-CQ #12)
)

// MaxRows caps the retained index (NFR-4 / SAM-D-22, P10-03): the most recent win.
const MaxRows = 500

// Row is one event in the index.
type Row struct {
	Key      string                // normalised id — the de-dup key
	Tab      Tab
	Source   string                // "USGS" | "NHC" | "NWS" | "NWS-local"
	Product  string                // "Tornado Warning" · "Earthquake" · "Tropical Storm"
	Name     string                // storm name; "" otherwise
	Location string                // the tied label (globalfeed.Locate) or the tracked location's label
	Tied     *snapshot.LocationRef // the tracked location the row belongs to; nil for a global-only row (the zone rule, FR-9)
	Severity globalfeed.Severity
	At       time.Time // declared / recorded / reported
	Until    time.Time // expires; zero when none
	HasPoint bool
	Lat, Lon float64
	Sender   string
	Sent     time.Time
	Detail   Detail
}

// Detail is the per-class record (SAM-D-21): exactly one is non-nil.
type Detail struct {
	Quake    *globalfeed.QuakeDetail
	Tropical *globalfeed.TropicalDetail
	Severe   *globalfeed.SevereDetail // the national feed's CAP record
	Alert    *snapshot.Alert          // the tracked-location CAP record (preferred when both exist)
}

// oidRE is the CAP alert identifier grammar api.weather.gov emits: a dotted
// OID with a hex fingerprint segment. Anything else keeps its raw id and never
// merges (red-team S4: a crafted "…/urn:oid:<real>" must not collide).
var oidRE = regexp.MustCompile(`^urn:oid:[0-9]+(\.[0-9a-f]+)+$`)

// NormalizeID joins the two forms an NWS alert id takes — the feature URL
// (https://api.weather.gov/alerts/urn:oid:…) on the ticker path and the bare
// urn:oid:… on the location path. nws reports whether the id validated as one.
func NormalizeID(id string) (key string, nws bool) {
	id = strings.TrimSpace(id)
	i := strings.Index(id, "urn:oid:")
	if i < 0 {
		return id, false
	}
	cand := id[i:]
	if !oidRE.MatchString(cand) {
		return id, false
	}
	if i > 0 && !strings.HasPrefix(id, "https://api.weather.gov/alerts/") {
		return id, false // an OID under a foreign prefix is not the same alert
	}
	return cand, true
}

// Classify maps an event class + product name to its tab; ok is false for a
// product the window does not show in v1 (Air Quality Alert, Hydrologic
// Outlook…). English substring matching on NWS product names — an accepted
// limitation (objectives §5).
func Classify(class globalfeed.Class, product string) (Tab, bool) {
	switch class {
	case globalfeed.ClassQuake:
		return TabQuakes, true
	case globalfeed.ClassTropical:
		return TabTropical, true
	}
	switch {
	case product == "Special Weather Statement":
		return TabStatements, true
	case strings.Contains(product, "Warning"):
		return TabWarnings, true
	case strings.Contains(product, "Watch"):
		return TabWatches, true
	case strings.Contains(product, "Advisory"):
		return TabAdvisories, true
	}
	return TabNone, false // "Statements" is Special Weather Statements by name (SAM-D-10); other *Statement products are not shown in v1
}

// Guard applies the guarded superseded rule to the tracked locations' alerts
// (NFR-12): the ids a same-sender, same-product, newer message replaces.
func Guard(locs []snapshot.Location) map[string]bool {
	var refs []globalfeed.Ref
	seen := map[string]bool{}
	for i := range locs {
		for _, a := range locs[i].Alerts {
			key, _ := NormalizeID(a.ID)
			if seen[key] {
				continue
			}
			seen[key] = true
			r := globalfeed.Ref{ID: key, Sender: a.SenderName, Product: a.Event, Sent: a.Sent}
			for _, x := range a.References {
				if k, _ := NormalizeID(x); k != key {
					r.Replaces = append(r.Replaces, k)
				}
			}
			refs = append(refs, r)
		}
	}
	return globalfeed.SupersededBy(refs)
}

// Union folds the feed events and the locations' alerts into one row per
// event: keyed on the normalised id; a tracked location's record wins over the
// feed's (it carries the prose), with the feed's point and tied label merged
// in; superseded messages on either path are dropped; a multi-location alert
// is one row tied to its first location in watchlist order; now drops expired.
func Union(feed []globalfeed.Event, locs []snapshot.Location, now time.Time) []Row {
	rows := make(map[string]Row)
	order := make([]string, 0, len(feed))
	add := func(r Row) {
		if _, ok := rows[r.Key]; !ok {
			order = append(order, r.Key)
		}
		rows[r.Key] = r
	}
	for _, e := range feed {
		if e.Superseded || e.ID == "" {
			continue
		}
		if !e.Until.IsZero() && now.After(e.Until) {
			continue
		}
		tab, ok := Classify(e.Class, e.Type)
		if !ok {
			continue
		}
		key, _ := NormalizeID(e.ID)
		r := Row{Key: key, Tab: tab, Source: e.Source, Product: e.Type, Name: e.Name, Location: e.Location,
			Severity: e.Severity, At: e.At, Until: e.Until, HasPoint: e.HasPoint, Lat: e.Lat, Lon: e.Lon,
			Detail: Detail{Quake: e.Quake, Tropical: e.Tropical, Severe: e.Severe}}
		if e.Severe != nil {
			r.Sender, r.Sent = e.Severe.SenderName, e.Severe.Sent
		}
		add(r)
	}
	superseded := Guard(locs)
	for _, e := range feed { // an alert the national feed already saw superseded must not resurface via a tracked location whose zone query lagged (red-team PLAN S2)
		if e.Superseded {
			key, _ := NormalizeID(e.ID)
			superseded[key] = true
		}
	}
	for i := range locs {
		loc := locs[i]
		ref := snapshot.LocationRef{Label: loc.Label, Tag: loc.Tag, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, TZ: loc.TZ}
		for j := range loc.Alerts {
			a := loc.Alerts[j]
			key, _ := NormalizeID(a.ID)
			if superseded[key] {
				continue
			}
			until := a.Expires
			if a.Ends != nil {
				until = *a.Ends
			}
			if !until.IsZero() && now.After(until) {
				continue
			}
			tab, ok := Classify(globalfeed.ClassSevereWx, a.Event)
			if !ok {
				continue
			}
			at := a.Effective
			if a.Onset != nil {
				at = *a.Onset
			}
			if at.IsZero() {
				at = a.Sent
			}
			alert := loc.Alerts[j]
			r := Row{Key: key, Tab: tab, Source: "NWS-local", Product: a.Event, Location: loc.Label, Tied: &ref,
				Severity: severityOf(a.Event, a.Severity), At: at, Until: until, Sender: a.SenderName, Sent: a.Sent,
				Detail: Detail{Alert: &alert}}
			if prev, ok := rows[key]; ok {
				if prev.Tied != nil {
					continue // already tied to an earlier (higher) watchlist location — one row per alert
				}
				r.HasPoint, r.Lat, r.Lon = prev.HasPoint, prev.Lat, prev.Lon // the feed's point
				r.Detail.Severe = prev.Detail.Severe                        // keep the national record's CAP parameters (gusts, hail, VTEC) beside the location record (red-team PLAN-CQ #5)
				if r.Severity < prev.Severity {
					r.Severity = prev.Severity // the curated tier is authoritative when the feed knows the product
				}
			}
			add(r)
		}
	}
	out := make([]Row, 0, len(order))
	for _, k := range order {
		out = append(out, rows[k])
	}
	return out
}

// severityOf maps a CAP severity + product to the tape's tier for a
// location-path alert the curated list does not know.
func severityOf(product, capSeverity string) globalfeed.Severity {
	switch strings.ToLower(capSeverity) {
	case "extreme":
		return globalfeed.SevRed
	case "severe":
		return globalfeed.SevOrange
	}
	if strings.Contains(product, "Warning") {
		return globalfeed.SevOrange
	}
	return globalfeed.SevYellow
}

// Sorted is Sort returning its argument (composable in tests).
func Sorted(rows []Row) []Row { Sort(rows); return rows }

// Sort orders rows Declared DESC (SAM-D-8), then most severe, then key — stable.
func Sort(rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].At.Equal(rows[j].At) {
			return rows[i].At.After(rows[j].At)
		}
		if rows[i].Severity != rows[j].Severity {
			return rows[i].Severity > rows[j].Severity
		}
		return rows[i].Key < rows[j].Key
	})
}

// Cap keeps the first n rows of a sorted slice (most recent wins) and reports
// the total before capping, for "showing N of M".
func Cap(rows []Row, n int) (kept []Row, total int) {
	total = len(rows)
	if len(rows) > n {
		rows = rows[:n]
	}
	return rows, total
}

// ByTab splits rows into their tabs, order preserved.
func ByTab(rows []Row) [NumTabs][]Row {
	var out [NumTabs][]Row
	for _, r := range rows {
		if r.Tab >= 0 && r.Tab < NumTabs {
			out[r.Tab] = append(out[r.Tab], r)
		}
	}
	return out
}
```

**Verify:** `go test ./domains/severe -run 'TestNormalizeID|TestClassify|TestUnion|TestSortAndCap' -v`

---

## Task 1.9 — `RecordOf`: the one class switch, `Plain` on every field

**File:** `domains/severe/record.go` (CREATE)

**Test first (RED):** `domains/severe/record_test.go`
```go
package severe

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

var chicago, _ = time.LoadLocation("America/Chicago")

func TestRecordOfWarningIsTheAlertShape(t *testing.T) {
	sent := time.Date(2026, 8, 28, 13, 45, 0, 0, time.UTC)
	ends := sent.Add(15 * time.Minute)
	r := Row{Key: "k", Tab: TabWarnings, Product: "Tornado Warning", At: sent, Until: ends, Detail: Detail{Alert: &snapshot.Alert{
		Event: "Tornado Warning", Severity: "extreme", Urgency: "immediate", Certainty: "observed", AreaDesc: "Johnson County, KS",
		SenderName: "NWS Kansas City", Description: "At 845 AM CDT, a severe thunderstorm…\n\n* HAZARD...Damaging tornado.", Instruction: "TAKE COVER NOW!",
	}}}
	rec := RecordOf(r, chicago)
	if rec.Title != "TORNADO WARNING" || rec.Meta != "[Extreme · Immediate · Observed]" {
		t.Fatalf("title/meta: %q %q", rec.Title, rec.Meta)
	}
	if rec.Timing != "Declared 08/28 08:45 CDT   Expires 08/28 09:00 CDT   (~15m)" {
		t.Fatalf("timing: %q", rec.Timing)
	}
	if rec.Area != "Area: Johnson County, KS · NWS Kansas City" || len(rec.Paras) != 3 || !strings.HasPrefix(rec.Paras[2], "Instructions: ") {
		t.Fatalf("area/paras: %q %v", rec.Area, rec.Paras)
	}
}

func TestRecordOfQuakeAndStorm(t *testing.T) {
	mag := 5.8
	cdi, mmi := 5.7, 4.624
	q := Row{Tab: TabQuakes, Product: "Earthquake", At: time.Date(2026, 8, 28, 10, 12, 0, 0, time.UTC), Location: "Kodāri, Nepal", Lat: 27.94, Lon: 85.62, HasPoint: true,
		Detail: Detail{Quake: &globalfeed.QuakeDetail{Mag: &mag, MagType: "mww", DepthKm: 61, Alert: "green", Felt: 153, CDI: &cdi, MMI: &mmi, Sig: 651, Status: "reviewed"}}}
	rec := RecordOf(q, time.UTC)
	if rec.Title != "M 5.8 EARTHQUAKE" || !strings.Contains(rec.Meta, "Depth 61 km") || !strings.Contains(rec.Area, "27.94 N, 85.62 E") {
		t.Fatalf("quake record: %+v", rec)
	}
	s := Row{Tab: TabTropical, Product: "Tropical Storm", Name: "Dolly", At: time.Date(2026, 8, 28, 16, 0, 0, 0, time.UTC),
		Detail: Detail{Tropical: &globalfeed.TropicalDetail{Name: "Dolly", Basin: "the Atlantic", BinNumber: "AT4", WindKt: 45, PressureMb: 999, MoveDirDeg: 280, MoveSpeedKt: 25, LatText: "15.0N", LonText: "46.9W", AdvisoryNum: "5"}}}
	rec = RecordOf(s, time.UTC)
	if rec.Title != "TROPICAL STORM DOLLY (AT4)" || !strings.Contains(rec.Meta, "Winds 45 kt") || !strings.Contains(rec.Meta, "Moving W at 25 kt") {
		t.Fatalf("storm record: %+v", rec)
	}
}

func TestRecordOfStripsTerminalEscapesFromEveryField(t *testing.T) {
	evil := "x\x1b]52;c;aGVsbG8=\x07y\x1b[31mz"
	r := Row{Tab: TabWarnings, Product: evil, Detail: Detail{Alert: &snapshot.Alert{Event: evil, Severity: evil, Urgency: evil, Certainty: evil, AreaDesc: evil, SenderName: evil, Headline: evil, Description: evil, Instruction: evil}}}
	rec := RecordOf(r, time.UTC)
	for _, s := range append([]string{rec.Title, rec.Meta, rec.Timing, rec.Area}, rec.Paras...) {
		if strings.ContainsAny(s, "\x1b\x07") {
			t.Fatalf("escape survived: %q", s)
		}
	}
}
```

**Code:**
```go
package severe

// record.go — RecordOf: the ONE place the per-class switch happens (AX-2 A,
// red-team C-18). Every field a feed supplied passes through render.Plain here,
// so no renderer can reintroduce an unstripped path (NFR-6, red-team S1).

import (
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/render"
)

// Record is the [A]-shaped presentation of one row: a title, a bracketed meta
// line, the timing line, the area/position line and unwrapped paragraphs.
type Record struct {
	Title  string
	Meta   string
	Timing string
	Area   string
	Paras  []string
}

// stamp is the record's clock: "08/28 08:45 CDT" in the given zone.
func stamp(t time.Time, in *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if in == nil {
		in = time.Local
	}
	return t.In(in).Format("01/02 15:04 MST")
}

// compass names a direction in degrees ("W", "NE").
func compass(deg int) string {
	names := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return names[((deg%360)+360)%360/45%8]
}

func title(s string) string { return strings.ToUpper(render.Plain(s)) }

func cap1(s string) string {
	s = render.Plain(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// RecordOf renders a row's record in the zone `in` (the tied location's clock,
// else the viewer's local one — FR-9).
func RecordOf(r Row, in *time.Location) Record {
	switch {
	case r.Detail.Alert != nil:
		return alertRecord(r, in)
	case r.Detail.Severe != nil:
		return severeRecord(r, in)
	case r.Detail.Quake != nil:
		return quakeRecord(r, in)
	case r.Detail.Tropical != nil:
		return tropicalRecord(r, in)
	}
	return Record{Title: title(r.Product), Timing: "Declared " + stamp(r.At, in), Area: "Area: " + render.Plain(r.Location)}
}

func timing(verb string, at, until time.Time, in *time.Location) string {
	s := verb + " " + stamp(at, in)
	if until.After(at) {
		s += "   Expires " + stamp(until, in) + "   (~" + shortDur(until.Sub(at)) + ")"
	}
	return s
}

// shortDur renders a window as "15m", "2h", "1h30m", "3d" — never "15m0s"
// (time.Duration's String keeps the zero seconds).
func shortDur(d time.Duration) string {
	d = d.Round(time.Minute)
	if d >= 24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours())/24)
	}
	h, m := int(d.Hours()), int(d.Minutes())%60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

func alertRecord(r Row, in *time.Location) Record {
	a := r.Detail.Alert
	rec := Record{
		Title:  title(a.Event),
		Meta:   "[" + cap1(a.Severity) + " · " + cap1(a.Urgency) + " · " + cap1(a.Certainty) + "]",
		Timing: timing("Declared", r.At, r.Until, in),
	}
	area := render.Plain(a.AreaDesc)
	if a.SenderName != "" {
		area += " · " + render.Plain(a.SenderName)
	}
	rec.Area = "Area: " + area
	rec.Paras = paragraphs(a.Description, a.Instruction)
	if extra := capExtras(r.Detail.Severe); extra != "" { // the national feed's CAP parameters, when the same alert came both ways
		rec.Paras = append([]string{extra}, rec.Paras...)
	}
	return rec
}

// capExtras renders the allowlisted CAP parameters as one line ("Wind gusts
// to 60 mph · Hail to 1.00 in"), "" when none.
func capExtras(d *globalfeed.SevereDetail) string {
	if d == nil {
		return ""
	}
	var extra []string
	if d.MaxWindGust != "" {
		extra = append(extra, "Wind gusts to "+render.Plain(d.MaxWindGust))
	}
	if d.MaxHailSize != "" {
		extra = append(extra, "Hail to "+render.Plain(d.MaxHailSize))
	}
	if d.EventMotion != "" {
		extra = append(extra, render.Plain(d.EventMotion))
	}
	return strings.Join(extra, " · ")
}

func severeRecord(r Row, in *time.Location) Record {
	d := r.Detail.Severe
	rec := Record{
		Title:  title(r.Product),
		Meta:   "[" + cap1(d.Severity) + " · " + cap1(d.Urgency) + " · " + cap1(d.Certainty) + "]",
		Timing: timing("Declared", r.At, r.Until, in),
	}
	area := render.Plain(r.Location)
	if d.SenderName != "" {
		area += " · " + render.Plain(d.SenderName)
	}
	rec.Area = "Area: " + area
	rec.Paras = paragraphs(d.Description, d.Instruction)
	if extra := capExtras(d); extra != "" {
		rec.Paras = append([]string{extra}, rec.Paras...)
	}
	return rec
}

func quakeRecord(r Row, in *time.Location) Record {
	q := r.Detail.Quake
	mag := "M ?"
	if q.Mag != nil {
		mag = fmt.Sprintf("M %.1f", *q.Mag)
	}
	meta := []string{}
	if q.Mag != nil {
		meta = append(meta, fmt.Sprintf("Magnitude %.1f %s", *q.Mag, render.Plain(q.MagType)))
	}
	if q.DepthKm > 0 {
		meta = append(meta, fmt.Sprintf("Depth %.0f km", q.DepthKm))
	}
	if q.Alert != "" {
		meta = append(meta, "PAGER "+q.Alert)
	}
	if q.Tsunami {
		meta = append(meta, "Tsunami yes")
	} else {
		meta = append(meta, "Tsunami no")
	}
	rec := Record{Title: mag + " " + title(r.Product), Meta: "[" + strings.Join(meta, " · ") + "]", Timing: "Recorded " + stamp(r.At, in)}
	if !q.UpdatedAt.IsZero() {
		rec.Timing += "   Updated " + stamp(q.UpdatedAt, in)
	}
	rec.Area = "Location: " + render.Plain(r.Location)
	if r.HasPoint {
		rec.Area += fmt.Sprintf(" (%.2f %s, %.2f %s)", abs(r.Lat), ns(r.Lat), abs(r.Lon), ew(r.Lon))
	}
	var facts []string
	if q.Felt > 0 {
		facts = append(facts, fmt.Sprintf("Felt reports %d", q.Felt))
	}
	if q.CDI != nil {
		facts = append(facts, fmt.Sprintf("Community intensity %.1f", *q.CDI))
	}
	if q.MMI != nil {
		facts = append(facts, fmt.Sprintf("Modelled intensity %.1f", *q.MMI))
	}
	if q.Sig > 0 {
		facts = append(facts, fmt.Sprintf("Significance %d", q.Sig))
	}
	if q.Status != "" {
		facts = append(facts, cap1(q.Status))
	}
	if len(facts) > 0 {
		rec.Paras = []string{strings.Join(facts, " · ")}
	}
	return rec
}

func tropicalRecord(r Row, in *time.Location) Record {
	d := r.Detail.Tropical
	head := title(r.Product)
	if d.Name != "" {
		head += " " + title(d.Name)
	}
	if d.BinNumber != "" {
		head += " (" + title(d.BinNumber) + ")"
	}
	meta := []string{}
	if d.WindKt > 0 {
		meta = append(meta, fmt.Sprintf("Winds %d kt", d.WindKt))
	}
	if d.PressureMb > 0 {
		meta = append(meta, fmt.Sprintf("Pressure %d mb", d.PressureMb))
	}
	if d.MoveSpeedKt > 0 {
		meta = append(meta, fmt.Sprintf("Moving %s at %d kt", compass(d.MoveDirDeg), d.MoveSpeedKt))
	}
	rec := Record{Title: head, Meta: "[" + strings.Join(meta, " · ") + "]", Timing: "Reported " + stamp(r.At, in)}
	if d.AdvisoryNum != "" {
		rec.Timing += "   Advisory " + render.Plain(d.AdvisoryNum)
		if !d.AdvisoryAt.IsZero() {
			rec.Timing += " issued " + stamp(d.AdvisoryAt, in)
		}
	}
	pos := "Position: " + render.Plain(d.LatText) + " " + render.Plain(d.LonText)
	if d.Basin != "" {
		pos += " · " + render.Plain(d.Basin) + " basin"
	}
	rec.Area = pos
	var adv []string
	if d.AdvisoryNum != "" {
		adv = append(adv, "public "+render.Plain(d.AdvisoryNum))
	}
	if d.ForecastNum != "" {
		adv = append(adv, "forecast "+render.Plain(d.ForecastNum))
	}
	if d.DiscussionNum != "" {
		adv = append(adv, "discussion "+render.Plain(d.DiscussionNum))
	}
	if len(adv) > 0 {
		rec.Paras = []string{"Advisories: " + strings.Join(adv, ", ") + " (nhc.noaa.gov)"} // named, never linked (S6)
	}
	return rec
}

// paragraphs splits CAP prose into paragraphs (blank-line separated), Plain'd,
// with the instruction as a final "Instructions: …" paragraph.
func paragraphs(description, instruction string) []string {
	var out []string
	for _, p := range strings.Split(render.Plain(description), "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if s := strings.TrimSpace(render.Plain(instruction)); s != "" {
		out = append(out, "Instructions: "+s)
	}
	return out
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
func ns(lat float64) string {
	if lat < 0 {
		return "S"
	}
	return "N"
}
func ew(lon float64) string {
	if lon < 0 {
		return "W"
	}
	return "E"
}
```
(`render.Plain` strips CSI/OSC/control sequences — the S-F6 owner; the test above is its positive control on
this path.)

**Verify:** `go test ./domains/severe -v`

---

## Task 1.10 — Fixtures and the COV table test

**Files:** `domains/globalfeed/testdata/*.json` — **already in place** (promoted at PLAN exit from the DISCOVER
probe; one home, red-team PLAN A-5). `domains/severe` tests read them via a relative path
(`../globalfeed/testdata/…`), never a copy.

**Steps:**
1. `ls domains/globalfeed/testdata/` — the four probe files are present; nothing to copy.
2. Add `domains/globalfeed/cov_test.go` — the machine-readable render list (FR-5 / M2):
```go
package globalfeed

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// TestRenderListCoverage is the COV metric (brief M2 v1.2.0): every field of
// the frozen render list (02-analysis/data-shape.md §4, column "Render v1")
// is populated by the parser from the committed probe samples.
func TestRenderListCoverage(t *testing.T) {
	q := fetchFixture(t, "usgs_significant_week.json", func(c *httpx.Client, base string) Source { return NewUSGS(c, base) })[0].Quake
	covQuake := map[string]bool{"Mag": q.Mag != nil, "MagType": q.MagType != "", "DepthKm": q.DepthKm != 0, "Title": q.Title != "", "Status": q.Status != "", "UpdatedAt": !q.UpdatedAt.IsZero(), "Sig": q.Sig != 0}
	s := fetchFixture(t, "nhc_currentstorms.json", func(c *httpx.Client, base string) Source { return NewNHC(c, base) })[0].Tropical
	covStorm := map[string]bool{"Name": s.Name != "", "WindKt": s.WindKt != 0, "PressureMb": s.PressureMb != 0, "MoveSpeedKt": s.MoveSpeedKt != 0, "LatText": s.LatText != "", "AdvisoryNum": s.AdvisoryNum != ""}
	var w *SevereDetail
	for _, e := range fetchFixture(t, "nws_active_unfiltered_trimmed.json", func(c *httpx.Client, base string) Source { return NewNWS(c, base) }) {
		if e.Type == "Extreme Heat Warning" {
			w = e.Severe
		}
	}
	covWarn := map[string]bool{"Headline": w.Headline != "", "Description": w.Description != "", "Instruction": w.Instruction != "", "Severity": w.Severity != "", "Urgency": w.Urgency != "", "Certainty": w.Certainty != "", "SenderName": w.SenderName != "", "Sent": !w.Sent.IsZero(), "Expires": !w.Expires.IsZero()}
	for name, cov := range map[string]map[string]bool{"quake": covQuake, "storm": covStorm, "warning": covWarn} {
		missing := 0
		for field, ok := range cov {
			if !ok {
				missing++
				t.Errorf("%s: render-list field %s not populated", name, field)
			}
		}
		t.Logf("COV %s: %d/%d", name, len(cov)-missing, len(cov))
	}
}
```
(`fetchFixture` is Task 1.3's helper in `detail_test.go`.) Note the PAGER `alert` and `felt`/`cdi`/`mmi` are nullable in the feed and are
asserted present only when the sample carries them — they are Render-when-present fields.

**Verify:** `go test ./domains/... ./platform/snapshot/... -count=1` — all green; `go vet ./...`;
`GOOS=linux go vet ./domains/...` (the Linux CI habit).

**Batch exit:** `make verify` green; commit `feat(severe): domain index, widened feeds, bounded alerts`.
