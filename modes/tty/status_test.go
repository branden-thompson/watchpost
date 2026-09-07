package tty

import (
	"fmt"
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
	head := strings.SplitN(v, "WATCHPOST Observer", 2)[1]
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
	for _, want := range []string{"Watchpost Status", "UPTIME", "API STATUS", "NWS", "PIPELINES", "PRIORITY", "RECENT", "ISSUES"} {
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
	out := strings.Join((Dashboard{}).issueLines(render.Opts{}, 0, sn, nil), "\n")
	rows := strings.Split(out, "\n")[1:] // past the column header
	if len(rows) == 0 || !strings.HasPrefix(strings.TrimSpace(rows[0]), "✘ NDBC") {
		t.Fatalf("provider errors must sort first:\n%s", out)
	}
	if !strings.Contains(out, "obs_stale") || !strings.Contains(out, "×3 (2 loc)") {
		t.Fatalf("stale warnings must fold into one class with counts:\n%s", out)
	}
	if strings.Count(out, "obs_stale") != 1 {
		t.Fatalf("one row per issue class:\n%s", out)
	}
	if (Dashboard{}).issueLines(render.Opts{}, 0, nil, nil)[0] != "   none" {
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
	// The REQUESTS block is gone: its counters are columns of the API STATUS
	// table now, keyed by the host they were counted against. The window they
	// cover is the UPTIME row at the top — printing it twice, three lines apart,
	// invited a reader to look for the difference (0.14.0).
	for _, want := range []string{"providers over", "ENDPOINT", "api.weather.gov", "1234", "980", "4560", "12.3M",
		"api.tidesandcurrents.noaa.gov", "PUBLISHES", "DUMPS", "last: 20260826T120000Z ok", "trigger: kill -USR1 4242"} {
		if !strings.Contains(v, want) {
			t.Fatalf("status modal missing %q:\n%s", want, v)
		}
	}
	// Before any traffic or dump: honest placeholders, never zeros dressed as data.
	quiet, _ := NewDashboard(Config{Version: "x", Stats: func() Stats { return Stats{} }})
	var qm tea.Model = quiet
	qm, _ = qm.Update(tea.WindowSizeMsg{Width: 133, Height: 60})
	qs, _ := qm.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	// REQUESTS is folded into PROVIDERS now, so there is ONE placeholder here.
	if qv := stripANSITest(qs.View().Content); !strings.Contains(qv, "none yet") || !strings.Contains(qv, "awaiting first snapshot") {
		t.Fatalf("both blocks must say what they are waiting for:\n%s", stripANSITest(qs.View().Content))
	}
	// No Stats hook (report-only wiring, tests): the sections are absent, not empty.
	plain := dash(t)
	ps, _ := plain.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if strings.Contains(ps.View().Content, "REQUESTS") {
		t.Fatal("without a Stats hook the modal must not show a REQUESTS section")
	}
}

// ONE COLUMN since 0.14.0: PROVIDERS used to sit beside REQUESTS, and REQUESTS
// is gone — its counters are columns of the providers table now, because they
// are per HOST and that is what the table is keyed by. Every remaining block is
// a wide table with nothing to pair it with. A blank line of air under the
// title; the sections in order; no line ever exceeds the terminal.
func TestStatusIsOneColumnAndFitsItsTerminal(t *testing.T) {
	stats := func() Stats {
		return Stats{
			Requests: httpx.RequestStats{Uptime: 10 * time.Minute, Hosts: []httpx.HostStats{
				{Host: "api.weather.gov", Attempts: 302, Net: 211, Cache: 206, BytesNet: 8_100_000},
				{Host: "earthquake.usgs.gov", Attempts: 14, Net: 14, Cache: 141, BytesNet: 77_200}}},
			Endpoints: map[string][]string{"nws": {"api.weather.gov"}},
			DumpHint:  "kill -USR1 29290 → /tmp/profiles",
		}
	}
	for _, w := range []int{100, 133, 200} {
		m, err := NewDashboard(Config{Version: "t", Stats: stats})
		if err != nil {
			t.Fatal(err)
		}
		var mm tea.Model = m
		mm, _ = mm.Update(tea.WindowSizeMsg{Width: w, Height: 44})
		mm, _ = mm.Update(SnapshotMsg{Snap: snap()})
		mm, _ = mm.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
		d := mm.(Dashboard)
		lines := d.statusLines()
		// The window's own row first — uptime and version — then air (0.14.0).
		if !strings.Contains(lines[0], "UPTIME") || lines[1] != "" {
			t.Fatalf("%d cols: the headline row then a blank, got %q / %q", w, lines[0], lines[1])
		}
		text := stripANSITest(strings.Join(lines, "\n"))
		for _, want := range []string{"API STATUS", "ENDPOINT", "api.weather.gov", "PIPELINES", "SEVERE INDEX", "ISSUES", "DUMPS", "kill -USR1"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%d cols: %q missing:\n%s", w, want, text)
			}
		}
		// A host NO PROVIDER CLAIMS still gets a row: httpx counts the ticker's
		// feeds and the geocoder, and dropping them would have this window
		// under-report the traffic it exists to account for.
		if !strings.Contains(text, "earthquake.usgs.gov") {
			t.Fatalf("%d cols: an unclaimed host must still be listed:\n%s", w, text)
		}
		last := -1
		for _, name := range []string{"API STATUS", "PIPELINES", "ISSUES", "DUMPS"} {
			if i := strings.Index(text, name); i < last {
				t.Fatalf("%d cols: %s out of order", w, name)
			} else {
				last = i
			}
		}
		for _, l := range strings.Split(stripANSITest(d.View().Content), "\n") {
			if render.Width(strings.TrimRight(l, " ")) > w {
				t.Fatalf("%d cols: a line runs past the terminal: %q", w, l)
			}
		}
	}
}

// AN UNMEASURED HOST IS NOT A FAILING ONE.
//
// The ticker's feeds and the geocoder are counted by httpx and are NOT snapshot
// providers, so they have no health to report. HealthGlyph reads anything that
// is not "ok" as a failure, which put a red ✘ on www.nhc.noaa.gov in [S] while
// the masthead — which counts snapshot providers only — said every provider was
// fine. Two surfaces contradicting each other about the same host is worse than
// either of them being silent.
func TestAnUnmeasuredHostReadsNeutralNotFailed(t *testing.T) {
	m, err := NewDashboard(Config{Version: "t", Stats: func() Stats {
		return Stats{
			Requests: httpx.RequestStats{Uptime: time.Hour, Hosts: []httpx.HostStats{
				{Host: "api.weather.gov", Attempts: 10, Net: 8},
				{Host: "www.nhc.noaa.gov", Attempts: 3, Net: 1}}},
			Endpoints: map[string][]string{"nws": {"api.weather.gov"}},
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	mm, _ = mm.Update(SnapshotMsg{Snap: snap()})
	d := mm.(Dashboard)

	var row string
	for _, l := range d.providerLines(d.opts(), d.cfg.Stats(), 0) {
		if strings.Contains(l, "www.nhc.noaa.gov") {
			row = stripANSITest(l)
		}
	}
	if row == "" {
		t.Fatal("a host with traffic must be listed even when no provider claims it")
	}
	if strings.Contains(row, "✘") {
		t.Errorf("an unmeasured host must not read as failed: %q", row)
	}
	if !strings.Contains(row, "not reported") {
		t.Errorf("it must say WHY it has no status: %q", row)
	}
	// And the masthead agrees: it counts providers, and this host is not one.
	if head := stripANSITest(d.header(d.opts())); !strings.Contains(head, "✘0") {
		t.Errorf("the masthead counts providers, so an unclaimed host must not appear as a failure: %q", head)
	}
}

// THE [S] TABLES ARE LAID OUT BY THE KIT, and the kit's contract is a
// RECTANGLE: every table line exactly the window's inner width, with no cell
// clipped to get there (HUM LEAD, UAT 2026-08-30 — "whenever we need a table, we
// use a go-studs table, that's why we vendored it").
//
// Both halves are pinned because the first migration satisfied one by breaking
// the other: the lines came back a cell over and were clamped, which squared the
// block off by eating "1m 30s" down to "1m 30", "1.5M" to "1.5" and the S off
// ROWS. A table that loses a character to stay tidy is worse than one that does
// not line up.
func TestStatusTablesAreRectanglesWithNothingClipped(t *testing.T) {
	for _, w := range []int{60, 68, 72, 80, 100, 133, 200} { // narrow widths included: 72 was where SEEN was clipped away
		d := benchDash(t, w, 44).(Dashboard)
		d.snap.Warnings = []snapshot.Warning{{
			Code: snapshot.WarnProviderError, Provider: "nws", Endpoint: "api.weather.gov",
			HTTPStatus: 502, Blame: snapshot.BlameProvider,
			Message: "alerts: https://api.weather.gov/alerts/active kept failing (last HTTP 502) after 2 attempts — provider degraded; serving last-good data",
		}}
		d.modal = modalStatus
		inner := d.statusInner()
		body := d.wrapModal(d.statusLines(), min(d.opts().Width, d.statusWidth()))
		for _, line := range body {
			if got := render.Width(stripANSITest(line)); got > inner {
				t.Fatalf("%d cols: a body line is %d wide, past the %d it has: %q", w, got, inner, stripANSITest(line))
			}
		}
		joined := strings.Join(body, "\n")
		// A heading and its cells arrive together or not at all: the tables drop a
		// column WHOLE when the room runs out, and never clip one.
		for _, want := range []string{"ENDPOINT", "STATUS", "FETCHED", "PROVIDER", "ERROR"} {
			if !strings.Contains(joined, want) {
				t.Fatalf("%d cols: the %s heading is not optional:\n%s", w, want, joined)
			}
		}
		// And the fill column does its job: the providers table reaches the
		// window's edge rather than ending short of it and leaving the window
		// lopsided on a wide terminal.
		for _, line := range body {
			plain := stripANSITest(line)
			if !strings.Contains(plain, "ENDPOINT") {
				continue
			}
			if got := render.Width(plain); got != inner {
				t.Fatalf("%d cols: the providers header is %d wide, want the full inner %d", w, got, inner)
			}
		}
	}
}

// A COLUMN IS SHOWN WHOLE OR NOT AT ALL (BUILD-exit red team, 2026-08-30).
//
// The ISSUES table had no width ladder, so at 72 columns the fixed columns
// summed past the window and the rectangle clamp ate the SEEN cell — the fold
// counts (×1, ×2 (2 loc)) simply vanished, with the heading gone too so nothing
// on screen said they had. A table that drops a column deliberately is honest;
// one that clips the last one is a table lying about how many columns it has.
func TestStatusIssueColumnsAreWholeOrAbsent(t *testing.T) {
	for _, w := range []int{60, 68, 72, 80, 100, 133} {
		d := benchDash(t, w, 40).(Dashboard)
		d.snap.Warnings = []snapshot.Warning{
			{Code: snapshot.WarnProviderError, Provider: "nws", Endpoint: "api.weather.gov",
				HTTPStatus: 502, Blame: snapshot.BlameProvider, Message: "alerts kept failing"},
			{Code: "obs_stale", Provider: "ndbc", Location: "A", Message: "stale"},
			{Code: "obs_stale", Provider: "ndbc", Location: "B", Message: "stale"},
		}
		d.modal = modalStatus
		body := strings.Join(d.wrapModal(d.statusLines(), min(d.opts().Width, d.statusWidth())), "\n")
		// Whatever survives, the FOLD COUNT and its heading agree: either both
		// are there or neither is, because a folded row without its count reads
		// as a single observation.
		if head, cell := strings.Contains(body, "SEEN"), strings.Contains(body, "×2 (2 loc)"); head != cell {
			t.Errorf("%d cols: SEEN heading=%v but its cell=%v — a half-shown column:\n%s", w, head, cell, body)
		}
		// The heading a row is read by is never optional.
		for _, must := range []string{"PROVIDER", "ENDPOINT", "ERROR"} {
			if !strings.Contains(body, must) {
				t.Errorf("%d cols: the %s column must survive every width:\n%s", w, must, body)
			}
		}
	}
}

// THE [S] WINDOW'S MEMO MUST BE ABLE TO HIT. Its key fingerprints the stats,
// which carry uptime as a nanosecond duration — fingerprinting that as it
// stands yields a different key every frame, so the most expensive window in
// the app rebuilds three tables on every tick for as long as it is open. The
// row shows whole seconds; anything finer is not a change a reader can see.
func TestStatusWindowMemoHitsBetweenVisibleChanges(t *testing.T) {
	d := benchDash(t, 133, 44).(Dashboard)
	held := time.Now()
	d.now = func() time.Time { return held }
	uptime := 3 * time.Hour
	d.cfg.Stats = func() Stats {
		uptime += 37 * time.Nanosecond // the clock runs, as it does live
		return Stats{Uptime: uptime, Version: "0.14.0", Requests: httpx.RequestStats{Uptime: uptime}}
	}
	d.modal = modalStatus
	_ = d.View().Content // the first render is the miss that fills the slot
	hitsBefore, _ := d.modalMemoCounts()
	const frames = 20
	for range frames {
		_ = d.View().Content
	}
	hitsAfter, missesAfter := d.modalMemoCounts()
	if got := hitsAfter - hitsBefore; got != frames {
		t.Errorf("%d of %d frames hit the memo (misses now %d) — nanosecond drift is defeating the key", got, frames, missesAfter)
	}
}

// THE ISSUES TABLE IS A FUNCTION OF ITS DATA. Collecting folded classes out of
// a map and sorting on a comparator that ties left the order to Go's random map
// iteration, so identical input rendered a different table each time — and the
// maxIssueRows cut then hid a different issue class on every render.
func TestIssueOrderIsStableAcrossRenders(t *testing.T) {
	snap := &snapshot.Snapshot{}
	for i := range 6 { // six classes, all the same count: every comparator ties
		snap.Warnings = append(snap.Warnings, snapshot.Warning{
			Code: fmt.Sprintf("class_%d", i), Provider: fmt.Sprintf("p%d", i),
			Message: "same", Endpoint: "example.test",
		})
	}
	first := issueOrder(foldWarnings([]*snapshot.Snapshot{snap}))
	for range 200 {
		if got := issueOrder(foldWarnings([]*snapshot.Snapshot{snap})); got != first {
			t.Fatalf("the same warnings must fold to the same order:\n first %q\n then  %q", first, got)
		}
	}
}

// issueOrder is the folded classes as one comparable string.
func issueOrder(issues []*issue) string {
	keys := make([]string, 0, len(issues))
	for _, it := range issues {
		keys = append(keys, string(it.code)+"|"+it.provider)
	}
	return strings.Join(keys, ",")
}

// A FOLDED ROW DESCRIBES ONE OCCURRENCE. The blame column is the point of the
// table, so pairing the first occurrence's status and blame with the last
// occurrence's message names the wrong side of a failure that has since changed.
func TestAFoldedIssueReportsOneOccurrenceThroughout(t *testing.T) {
	snap := &snapshot.Snapshot{Warnings: []snapshot.Warning{
		{Code: snapshot.WarnProviderError, Provider: "nws", Endpoint: "api.weather.gov",
			HTTPStatus: 502, Blame: snapshot.BlameProvider, Message: "gateway failed"},
		{Code: snapshot.WarnProviderError, Provider: "nws", Endpoint: "api.weather.gov",
			HTTPStatus: 404, Blame: snapshot.BlameWatchpost, Message: "zone not found"},
	}}
	folded := foldWarnings([]*snapshot.Snapshot{snap})
	if len(folded) != 1 {
		t.Fatalf("both warnings are one class, got %d", len(folded))
	}
	got := folded[0]
	if got.latest != "zone not found" {
		t.Errorf("the message is the latest occurrence's, got %q", got.latest)
	}
	if got.status != 404 || got.blame != snapshot.BlameWatchpost {
		t.Errorf("the status and blame must describe the SAME occurrence as the message, got HTTP %d blame %q", got.status, got.blame)
	}
}

// A CELL'S TONE TRAVELS WITH THE CELL. The providers table's column set varies
// by form — PROVIDERS is dropped on a narrow window — so a tone assigned by
// POSITION lands on a different column at different widths. Muting "the second
// cell" for an unreported host muted PROVIDERS at the wide forms and STATUS at
// the narrow one, and no test covered the width where it bit.
func TestUnreportedHostsReadMutedAtEveryWidth(t *testing.T) {
	unreported := endpointRow{endpoint: "www.nhc.noaa.gov", providers: "—", state: "not reported"}
	muted := render.Tok(render.TableMuted)
	for form := range statusForms {
		cells := endpointCells(render.Opts{Width: 133}, unreported, form)
		for i, c := range cells {
			if i == 0 {
				continue // the endpoint wears its own health mark's tone
			}
			if c.tone != muted {
				t.Errorf("form %d cell %d (%q) reads %q, want the muted tone: a host nobody vouches for must not look reported", form, i, c.text, c.tone)
			}
		}
	}
	// And a host that IS reported keeps the base tone, so the distinction means
	// something.
	reported := endpointRow{endpoint: "api.weather.gov", providers: "NWS", state: "REF OK", claimed: true}
	if got := endpointCells(render.Opts{Width: 133}, reported, 0)[1].tone; got == muted {
		t.Errorf("a reported host must not read muted, got %q", got)
	}
}
