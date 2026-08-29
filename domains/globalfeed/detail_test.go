package globalfeed

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
	if clampNonNeg(999, 250) != 250 || clampNonNeg(-5, 250) != 0 || clampNonNeg(45, 250) != 45 {
		t.Fatal("clampNonNeg bounds wrong")
	}
	s := make([]string, maxListLen+10)
	if got := clampSlice(s); len(got) != maxListLen {
		t.Fatalf("clampSlice kept %d, want %d", len(got), maxListLen)
	}
}

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
	// A cache hit never decodes, even before any hash is known to differ.
	hits := 0
	cached := func() ([]byte, bool, error) { hits++; return body, true, nil }
	if _, _ = m.events(cached, parse); decodes != 2 {
		t.Fatalf("a cache hit re-decoded: %d", decodes)
	}
	// An error is never memoised: the next call parses again.
	bad := func([]byte) ([]Event, error) { return nil, os.ErrInvalid }
	body = []byte(`{"a":3}`)
	if _, err := m.events(get, bad); err == nil {
		t.Fatal("parse error swallowed")
	}
	if _, _ = m.events(get, parse); decodes != 3 {
		t.Fatalf("after an error the body must be re-parsed: %d", decodes)
	}
}

func TestSourceMemoHandsOutItsOwnSlice(t *testing.T) {
	m := &sourceMemo{}
	get := func() ([]byte, bool, error) { return []byte("b"), false, nil }
	parse := func([]byte) ([]Event, error) { return []Event{{ID: "x"}}, nil }
	a, _ := m.events(get, parse)
	a[0].Location = "written by a caller"
	b, _ := m.events(get, parse)
	if b[0].Location != "" {
		t.Fatal("the memo's events leaked a caller's write (Locate writes into elements)")
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

func TestUSGSDecodesTheRenderList(t *testing.T) {
	evs := fetchFixture(t, "usgs_significant_week.json", func(c *httpx.Client, base string) Source { return NewUSGS(c, base) })
	if len(evs) == 0 {
		t.Fatal("no events from the fixture")
	}
	deep := false
	for _, e := range evs {
		q := e.Quake
		if q == nil {
			t.Fatalf("Quake detail nil: %+v", e)
		}
		// COV: every field in the render list is populated from the fixture
		// (data-shape.md §4). Depth is 0 for a surface event (the sample's
		// landslide), so it is asserted across the set, not per event.
		if q.Mag == nil || q.MagType == "" || q.Title == "" || q.Status == "" || q.UpdatedAt.IsZero() || q.Sig == 0 {
			t.Fatalf("render list incomplete: %+v", *q)
		}
		if q.DepthKm > 0 {
			deep = true
		}
		if e.Name != "" {
			t.Fatal("a quake has no name")
		}
	}
	if !deep {
		t.Fatal("no event carried a depth")
	}
}

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
	anyParam := false
	for _, e := range evs {
		if e.Severe != nil && (e.Severe.MaxWindGust != "" || e.Severe.NWSHeadline != "" || e.Severe.VTEC != "") {
			anyParam = true
		}
	}
	if !anyParam {
		t.Fatal("no allowlisted parameter decoded (the fixture carries maxWindGust/NWSheadline/VTEC)")
	}
}

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
