package tty

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestStatusModalAndControlPlacement(t *testing.T) {
	// UAT 24: [S] floats API diagnostics; [+/-] lives in the player line,
	// not the footer; header reads 'Last Updated:' with the [S] chip.
	m := dash(t)
	v := m.View().Content
	if !strings.Contains(v, "Updated:") || strings.Contains(v, "DATA LAST UPDATED") {
		t.Fatalf("header wording (UAT 24.3):\n%s", v)
	}
	head := strings.SplitN(v, "W A T C H P O S T", 2)[1]
	if first := head; !strings.Contains(first, "Status") || !strings.Contains(first, "API: ") {
		t.Fatal("the title line carries the API summary and the [S] Status chip (UAT 24.2 / 102)")
	}
	if !strings.Contains(v, "VOL") || !strings.Contains(v, "[-]") || !strings.Contains(v, "[+]") {
		t.Fatal("volume control must render VOL [-]bar[+] in the player (UAT 41)")
	}
	if strings.Count(v, "Adjust Radio Volume") != 0 {
		t.Fatal("the [+/-] chip is gone (UAT 41)")
	}
	m2, _ := m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	sv := m2.View().Content
	for _, want := range []string{"API Status", "PROVIDERS", "NWS", "PIPELINES", "Priority", "Recent", "ISSUES"} {
		if !strings.Contains(sv, want) {
			t.Fatalf("status modal missing %q:\n%s", want, sv)
		}
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if strings.Contains(m3.View().Content, "PROVIDERS") {
		t.Fatal("esc must close the status modal")
	}
}

func TestStatusModalWrapsNeverTruncates(t *testing.T) {
	// UAT 25 (the recurring class, now fixed in the component): every modal
	// body line wraps within the tile — no … anywhere in the modal. The
	// longest line today is the dump trigger's path (quality pass Q0).
	long := "kill -USR1 4242 → /Users/someone/Library/Caches/watchpost/profiles/and-a-deeper-directory"
	m, err := NewDashboard(Config{Version: "x", Stats: func() Stats { return Stats{DumpHint: long} }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	m2, _ := model.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	v := m2.View().Content
	if !strings.Contains(v, "and-a-deeper-directory") {
		t.Fatalf("long diagnostic line must survive by wrapping:\n%s", v)
	}
	if strings.Contains(v, "…") {
		t.Fatalf("modal content must never truncate:\n%s", v)
	}
}

func TestStatusAlignmentAndIssueAggregation(t *testing.T) {
	// UAT 31.1: fixed-width ages line up; warnings fold into issue classes.
	for dur, want := range map[time.Duration]string{
		59*time.Minute + 59*time.Second: "59m 59s",
		1*time.Minute + 5*time.Second:   " 1m  5s",
		55 * time.Second:                "    55s",
		2*time.Hour + 5*time.Minute:     " 2h 05m",
	} {
		if got := fixedAge(dur); got != want {
			t.Fatalf("fixedAge(%v) = %q, want %q", dur, got, want)
		}
	}
	sn := &snapshot.Snapshot{Warnings: []snapshot.Warning{
		{Code: snapshot.WarnObsStale, Provider: "nws", Location: "A", Message: "obs 2h old"},
		{Code: snapshot.WarnObsStale, Provider: "nws", Location: "B", Message: "obs 3h old"},
		{Code: snapshot.WarnObsStale, Provider: "nws", Location: "B", Message: "obs 3h old again"},
		{Code: snapshot.WarnProviderError, Provider: "ndbc", Message: "cannot reach ndbc"},
	}}
	out := strings.Join(aggregateWarnings(sn, nil), "\n")
	if !strings.HasPrefix(strings.TrimSpace(out), "✘ NDBC") {
		t.Fatalf("provider errors must sort first:\n%s", out)
	}
	if !strings.Contains(out, "obs_stale") || !strings.Contains(out, "×3 (2 locations)") {
		t.Fatalf("stale warnings must fold into one class with counts:\n%s", out)
	}
	if strings.Count(out, "obs_stale") != 1 {
		t.Fatalf("one row per issue class:\n%s", out)
	}
	if aggregateWarnings(nil, nil)[0] != "   none" {
		t.Fatal("no warnings must read none")
	}
}

// Quality pass Q0 (plan Q0 task 3): the [S] modal carries the request
// counters per host, the publish counters per pipeline, and the last
// diagnostic dump — and says "none yet" honestly before any traffic.
func TestStatusModalShowsRequestAndDumpRows(t *testing.T) {
	stats := func() Stats {
		return Stats{
			Requests: httpx.RequestStats{Uptime: 2*time.Hour + 5*time.Minute, Hosts: []httpx.HostStats{
				{Host: "api.weather.gov", Attempts: 1234, Net: 980, Cache: 4560, Neg: 3, BytesNet: 12_900_000},
				{Host: "api.tidesandcurrents.noaa.gov", Attempts: 40, Net: 40}}},
			Pipelines: [2]PipelineStats{{Publishes: 17, Folded: 5}},
			LastDump:  "20260826T120000Z ok /tmp/profiles/20260826T120000Z",
			DumpHint:  "kill -USR1 4242 → /tmp/profiles",
		}
	}
	m, err := NewDashboard(Config{Version: "x", Stats: stats})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 60})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	sv, _ := model.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	v := stripANSITest(sv.View().Content)
	for _, want := range []string{"REQUESTS (since launch,  2h 05m)", "HOST", "api.weather.gov", "1234", "980", "4560", "12.3M",
		"api.tidesandcurrents.noaa.gov", "17 publishes (5 folded)", "DUMPS", "last: 20260826T120000Z ok", "trigger: kill -USR1 4242"} {
		if !strings.Contains(v, want) {
			t.Fatalf("status modal missing %q:\n%s", want, v)
		}
	}
	// Before any traffic or dump: honest placeholders, never zeros dressed as data.
	quiet, _ := NewDashboard(Config{Version: "x", Stats: func() Stats { return Stats{} }})
	var qm tea.Model = quiet
	qm, _ = qm.Update(tea.WindowSizeMsg{Width: 133, Height: 60})
	qs, _ := qm.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if qv := stripANSITest(qs.View().Content); strings.Count(qv, "none yet") != 2 {
		t.Fatalf("REQUESTS and DUMPS must each read 'none yet' before traffic:\n%s", qv)
	}
	// No Stats hook (report-only wiring, tests): the sections are absent, not empty.
	plain := dash(t)
	ps, _ := plain.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if strings.Contains(ps.View().Content, "REQUESTS") {
		t.Fatal("without a Stats hook the modal must not show a REQUESTS section")
	}
}

// The [S] window lays PROVIDERS beside REQUESTS when the terminal is wide
// enough (HUM LEAD UAT 2026-08-28) and stacks them when it is not; a blank
// line of air under the title in both; PIPELINES, ISSUES and DUMPS follow
// full width; no line ever exceeds the terminal.
func TestStatusLaysOutOneOrTwoColumns(t *testing.T) {
	stats := func() Stats {
		return Stats{Requests: httpx.RequestStats{Uptime: 10 * time.Minute, Hosts: []httpx.HostStats{{Host: "api.weather.gov", Attempts: 302, Net: 211, Cache: 206, BytesNet: 8_100_000}, {Host: "earthquake.usgs.gov", Attempts: 14, Net: 14, Cache: 141, BytesNet: 77_200}}}, DumpHint: "kill -USR1 29290 → /tmp/profiles"}
	}
	for _, c := range []struct {
		w      int
		twoCol bool
	}{{100, false}, {133, true}, {200, true}} {
		m, err := NewDashboard(Config{Version: "t", Stats: stats})
		if err != nil {
			t.Fatal(err)
		}
		var mm tea.Model = m
		mm, _ = mm.Update(tea.WindowSizeMsg{Width: c.w, Height: 44})
		mm, _ = mm.Update(SnapshotMsg{Snap: snap()})
		mm, _ = mm.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
		d := mm.(Dashboard)
		lines := d.statusLines()
		if lines[0] != "" {
			t.Fatalf("%d cols: a blank line under the title, got %q", c.w, lines[0])
		}
		text := stripANSITest(strings.Join(lines, "\n"))
		pair := false
		for _, l := range strings.Split(text, "\n") {
			if strings.Contains(l, "PROVIDERS") && strings.Contains(l, "REQUESTS") {
				pair = true
			}
		}
		if pair != c.twoCol {
			t.Fatalf("%d cols: two columns = %v, want %v:\n%s", c.w, pair, c.twoCol, text)
		}
		for _, want := range []string{"PROVIDERS", "REQUESTS", "api.weather.gov", "PIPELINES", "severe index", "ISSUES", "DUMPS", "kill -USR1"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%d cols: %q missing:\n%s", c.w, want, text)
			}
		}
		order := []string{"PIPELINES", "ISSUES", "DUMPS"} // the full-width sections keep their order below the columns
		last := -1
		for _, name := range order {
			if i := strings.Index(text, name); i < last {
				t.Fatalf("%d cols: %s out of order", c.w, name)
			} else {
				last = i
			}
		}
		for _, l := range strings.Split(stripANSITest(d.View().Content), "\n") {
			if render.Width(strings.TrimRight(l, " ")) > c.w {
				t.Fatalf("%d cols: a line overflows the terminal: %q", c.w, l)
			}
		}
	}
}
