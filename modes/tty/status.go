package tty

// status.go — the [S] Watchpost Status window: uptime and version, the endpoint table, the pipelines,
// the issues and the dumps. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// statusLines is the [S] API diagnostics body: three tables — the endpoints and
// their request counters, the pipelines, and the issues — plus the dumps.
//
// ONE COLUMN since 0.14.0. PROVIDERS used to sit beside REQUESTS, and REQUESTS
// is gone: its counters are columns of the providers table now, because httpx
// counts per HOST and that is what the table is keyed by (HUM LEAD, UAT
// 2026-08-30). Every remaining block is a wide table with nothing to pair it
// with.
func (d Dashboard) statusLines() []string {
	o := d.opts()
	// Built twice: once at NATURAL width to decide how wide the window wants to
	// be, then again filled to it. The fill target depends on the window and the
	// window depends on the content, so one of the two passes has to be
	// unfilled — and the modal memo means a frame does this once, not per tick.
	blocks := d.statusBlocks(d.statusInner())
	lines := []string{d.statusHeadline(o), ""} // the window's own row, then air
	lines = append(lines, stacked(blocks.providers, blocks.pipelines, blocks.issues, blocks.dumps)...)
	return append(lines, "", " "+o.Controls("   ", render.Ctl("esc", "Close"), render.Ctl("↑↓", "Scroll")))
}

// statusWidth is the window's width: what its widest table needs, floored at the
// stretch every content-heavy window uses (UAT 31.2: 60 % of the terminal, 68
// the floor) and bounded by the terminal.
//
// CONTENT-SIZED, and it has to be. These are tables whose columns are padded to
// line up, and the panel WRAPS a line it cannot fit — a wrap re-flows on
// whitespace, so an over-wide table row comes back with every column collapsed
// to a single space. A window too narrow for its table does not truncate it, it
// destroys it.
func (d Dashboard) statusWidth() int {
	b := d.statusBlocks(0) // natural: what the content wants before any stretch
	want := widest(b.providers, b.pipelines, b.issues, b.dumps) + panelFrame + panelRail + columnMargin
	return min(d.opts().Width, max(max(68, d.width*60/100), want))
}

// statusInner is the content width inside the window's frame and its scroll
// rail — what a table may fill, and what the headline row is padded to, so the
// two end on the same column.
func (d Dashboard) statusInner() int {
	return min(d.opts().Width, d.statusWidth()) - panelFrame - panelRail
}

// statusHeadline is the window's own row: how long this run has been up on the
// left, what it is and whether there is a newer one on the right.
//
// The version reads UP-TO-DATE only when the check has actually answered. Until
// then, and whenever it cannot reach GitHub, it says nothing about staleness
// rather than claiming freshness it has not confirmed — a dashboard that told a
// listener they were current because it failed to ask would be worse than one
// that never checked (0.14.0).
func (d Dashboard) statusHeadline(o render.Opts) string {
	st := d.stats()
	left := "   UPTIME - " + longAge(st.Uptime)
	ver := "v" + st.Version
	var note string
	switch {
	case st.Behind:
		note = render.Tint(o.Glyphs().Alert+" v"+st.Latest+" available", render.Tok(render.AlertLabel))
	case st.Latest != "":
		note = render.Tint(o.Glyphs().OK+" up to date", render.Tok(render.ProviderOK))
	case st.CheckEnabled:
		note = render.Tint("· update check pending", render.Tok(render.TableMuted))
	}
	// The note is the first thing to go on a narrow window: the version names
	// the build, the note only qualifies it. Without the ladder the row ran past
	// the window's inner width and the panel re-flowed it.
	inner := d.statusInner()
	room := max(1, inner-render.Width(left)-1)
	right := render.FirstFit(room, ver+"   "+note, ver, "")
	if note == "" {
		right = render.FirstFit(room, ver, "")
	}
	return render.PadBetween(left, right, inner)
}

// longAge is an uptime a person reads: "3d 04h 12m 09s", dropping the leading
// units that are zero so a fresh run is "12m 09s" rather than "0d 00h 12m 09s".
func longAge(d time.Duration) string {
	if d <= 0 {
		return "just started"
	}
	d = d.Round(time.Second)
	days, hours := int(d/(24*time.Hour)), int(d/time.Hour)%24
	mins, secs := int(d/time.Minute)%60, int(d/time.Second)%60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %02dh %02dm %02ds", days, hours, mins, secs)
	case hours > 0:
		return fmt.Sprintf("%dh %02dm %02ds", hours, mins, secs)
	case mins > 0:
		return fmt.Sprintf("%dm %02ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// stats reads the app's stats hook once, with a zero value when there is none.
func (d Dashboard) stats() Stats {
	if d.cfg.Stats == nil {
		return Stats{}
	}
	return d.cfg.Stats()
}

// statusHeader is a section title: bold white at the 1-cell inset, as the
// Help window's group titles.
func statusHeader(name string) string { return " " + render.Tint(name, render.Tok(render.ModalTitle)) }

// statusSections are the [S] body's blocks, each a header and its rows.
//
// [S] answers "is the DATA arriving?". Who is reading and which tones sound are
// settings, and 0.14.0 briefly reported them here as well — two windows telling
// the same story, one of which could not change it. They live in [s] Settings,
// which is the window that owns them.
type statusSections struct {
	providers, pipelines, issues, dumps []string
}

// statusBlocks composes the sections (headers inset 1, rows inset 3 — with
// the panel's own 2, a 3 / 5 inset, HUM LEAD UAT 2026-08-28): aligned
// provider rows with a fixed-width freshness age, the request counters per host, pipeline
// snapshot ages, warnings AGGREGATED by code+provider, the dumps.
func (d Dashboard) statusBlocks(fillTo int) statusSections {
	o := d.opts()
	var b statusSections
	var st Stats
	if d.cfg.Stats != nil {
		st = d.cfg.Stats()
		b.dumps = dumpLines(st)
	}
	b.providers = d.providerLines(o, st, fillTo)
	b.pipelines = d.pipelineLines(o, st, fillTo)
	b.issues = append([]string{statusHeader("ISSUES")}, d.issueLines(o, fillTo, d.snap, d.recent)...)
	return b
}

// providerLines is the PROVIDERS table: ONE ROW PER ENDPOINT, naming the
// providers that use it, its freshness, and the request counters for that host.
//
// Keyed by endpoint because the counters are: httpx counts per HOST, and
// several providers share one — nws and nws-marine are both api.weather.gov.
// A row per provider would either repeat the same numbers twice or leave them
// blank, and neither says what is true.
func (d Dashboard) providerLines(o render.Opts, st Stats, fillTo int) []string {
	rows := endpointRows(providersOf(d.snap), st)
	// THE TWO COUNTS RECONCILED, on the header. The masthead counts PROVIDERS
	// and this table has one row per ENDPOINT — nws and nws-marine share
	// api.weather.gov, coops and coops-obs share tidesandcurrents — so a
	// listener counting rows finds fewer than the masthead promised and has no
	// way to see why. The header says both.
	// NO "requests since launch, <time>" HERE. The counters are cumulative from
	// launch and the window they cover is the UPTIME on the row above, which is
	// the same value — printing it twice, three lines apart, invites a reader to
	// look for the difference between them.
	lines := []string{statusHeader(fmt.Sprintf("API STATUS  (%d providers over %d endpoints)",
		len(providersOf(d.snap)), claimedRows(rows)))}
	for i := range rows { // the age is read ONCE, so every row of a frame agrees
		if !rows[i].fetchedAt.IsZero() {
			rows[i].age = d.now().Sub(rows[i].fetchedAt)
		}
	}
	if len(rows) == 0 {
		return append(lines, "   awaiting first snapshot...")
	}
	// A LADDER, widest form first, exactly as the masthead's control row does.
	// The window can never be wider than the terminal, and the panel WRAPS what
	// it cannot fit — a wrap re-flows on whitespace, so an over-wide table row
	// comes back with every column collapsed to one space. A table that will not
	// fit has to lose columns, not alignment.
	for form := range statusForms {
		if out, ok := providerTable(o, rows, form, statusAvail(o), fillTo); ok {
			return append(lines, out...)
		}
	}
	out, _ := providerTable(o, rows, statusForms-1, 0, fillTo) // the narrowest, whatever the room
	return append(lines, out...)
}

// claimedRows is how many rows a PROVIDER stands behind — the ticker's feeds
// and the geocoder are listed for their traffic but are not providers, so they
// do not count toward the masthead's total.
func claimedRows(rows []endpointRow) int {
	n := 0
	for _, r := range rows {
		if r.claimed {
			n++
		}
	}
	return n
}

// endpointMark is the row's health glyph — or a NEUTRAL DASH when no snapshot
// provider reports on that host.
//
// The ticker's feeds and the geocoder are counted by httpx and are not snapshot
// providers, so they have no status to show. HealthGlyph reads anything that is
// not "ok" as a failure, which put a red ✘ on www.nhc.noaa.gov while the
// masthead — which counts snapshot providers only — said everything was fine
// . An unmeasured host is not a failing one, and the
// two surfaces must not contradict each other about it.
func endpointMark(o render.Opts, r endpointRow) (glyph, tone string) {
	// PLAIN TEXT and a separate tone, never a pre-tinted cell. The kit measures
	// a cell by its BYTES, so an escape sequence inside one is counted as
	// content and the column comes out mangled — CellStyles exists so the tone
	// is applied after the padding, which is the only order that can be right.
	status := r.status
	if !r.claimed {
		status = snapshot.ProviderOff
	}
	switch {
	case !r.claimed:
		return o.Glyphs().Dash, render.Tok(render.TextBase)
	case status == snapshot.ProviderOK:
		return o.Glyphs().OK, render.Tok(render.ProviderOK)
	}
	return o.Glyphs().Fail, render.Tok(render.ProviderDown)
}

// statusForms is how many widths the providers table has: everything, then the
// counters thinned, then dropped, then the providers column with them.
const statusForms = 4

// statusAvail is the width a [S] table has to work in — the terminal's content
// width less the panel's chrome.
//
// Measured from the TERMINAL, not the window: the window's width is computed
// FROM these lines, so asking it would be a circle (the same reason the tone
// group measures the terminal).
func statusAvail(o render.Opts) int { return o.Width - panelFrame - panelRail - columnMargin }

// providerTable renders the table at one form, and reports whether it fits.
//
// THE KIT LAYS IT OUT (platform/render's StatusTable): ENDPOINT is the FILL
// column, so the table spans the window's inner width instead of ending short of
// it, and it is truncatable, so a host longer than the room is cut rather than
// pushing the measurements out of line. Everything else is a fixed width — a
// number and its heading must not drift apart.
func providerTable(o render.Opts, rows []endpointRow, form, avail, fillTo int) ([]string, bool) {
	provW, stateW := len("PROVIDERS"), len("STATUS")
	epMin := len("ENDPOINT")
	for _, r := range rows {
		provW, stateW = max(provW, render.Width(r.providers)), max(stateW, render.Width(r.state))
		epMin = min(max(epMin, render.Width(r.endpoint)), statusEndpointMax)
	}
	// THE MARK RIDES IN THE ENDPOINT CELL rather than a column of its own.
	//
	// ✔ is an ambiguous-width rune: the kit's width table reads it as two cells
	// where ours reads one, so a fixed column sized for it came out a cell adrift
	// on every row. The fix is not to argue with the table about a rune, it is to
	// leave the disagreement nowhere to land — inside the FILL column the kit can
	// only ever under-pad, which PadTo corrects, and over-running is the failure
	// that actually costs content.
	//
	// It reads better too: the endpoint carries its own health tone, so a failing
	// host is a red NAME rather than a red mark beside a grey one.
	cols := []render.StatusColumn{
		{Header: statusInset + strings.Repeat(" ", statusMarkW) + "ENDPOINT", Fill: true,
			MinWidth: len(statusInset) + statusMarkW + epMin, Truncatable: true},
	}
	if form < 3 {
		cols = append(cols, render.StatusColumn{Header: "PROVIDERS", Width: provW, Truncatable: true, MinWidth: len("PROVIDERS")})
	}
	cols = append(cols,
		render.StatusColumn{Header: "STATUS", Width: stateW},
		render.StatusColumn{Header: "FETCHED", Width: 8, Right: true})
	switch form {
	case 0:
		cols = append(cols, counterCols("TRIES", "NET", "CACHE", "NEG", "BYTES")...)
	case 1:
		cols = append(cols, counterCols("TRIES", "NET", "CACHE")...)
	}

	out := make([]render.StatusRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, endpointCells(o, r, form).row())
	}
	// FIT IS TESTED AT THE NATURAL WIDTH, not the filled one. A filled table is
	// exactly as wide as it was told to be, so testing that against the room
	// always says "too wide" and the ladder falls straight to its narrowest
	// form — which is what happened the first time.
	head := render.Tok(render.ModalTitle)
	natural := o.StatusTable(cols, out, render.StatusNaturalWidth(cols), head)
	if widest(natural) > avail {
		return natural, false
	}
	// THE MEASURING PASS RETURNS THE NATURAL TABLE. Filling to the room
	// available on that pass would tell statusWidth the content wants the whole
	// terminal, and the window would stretch to 193 columns on a 200-column
	// screen instead of the 60 % every content window uses.
	if fillTo > 0 {
		return o.StatusTable(cols, out, fillTo, head), true
	}
	return natural, true
}

// tonedCell is one cell together with the tone it is painted in, so a cell and
// its colour are carried as one thing.
//
// The alternative — a slice of strings beside a map keyed by position — makes
// the colour depend on where a column happens to sit, and this table's column
// set VARIES BY FORM. Muting "the second cell" meant muting PROVIDERS at the
// wide forms and STATUS at the narrow one, which is the width the app's own
// supported floor sits at.
type tonedCell struct {
	text string
	tone string
}

// tonedRow is one table row.
type tonedRow []tonedCell

// row is the pair render.StatusTable wants, derived rather than maintained.
func (cells tonedRow) row() render.StatusRow {
	out := render.StatusRow{Cells: make([]string, 0, len(cells)), Styles: map[int]string{}}
	for i, c := range cells {
		out.Cells = append(out.Cells, c.text)
		out.Styles[i] = c.tone
	}
	return out
}

// endpointCells is one endpoint's row, in the column order providerTable builds
// for this form.
func endpointCells(o render.Opts, r endpointRow, form int) tonedRow {
	text, muted := render.Tok(render.TextBase), render.Tok(render.TableMuted)
	// A host no snapshot provider reports on reads quieter throughout: its
	// numbers are real, but nothing is vouching for its health.
	if !r.claimed {
		text = muted
	}
	age := "n/a"
	if !r.fetchedAt.IsZero() {
		age = strings.TrimSpace(fixedAge(r.age))
	}
	mark, markTone := endpointMark(o, r)
	cells := tonedRow{{statusInset + render.PadTo(mark, statusMarkW) + r.endpoint, markTone}}
	if form < 3 {
		cells = append(cells, tonedCell{r.providers, text})
	}
	cells = append(cells, tonedCell{r.state, text}, tonedCell{age, text})
	for _, counter := range r.counters(form) {
		cells = append(cells, tonedCell{counter, text})
	}
	return cells
}

// statusMarkW is the health mark's cell: the glyph and the space after it.
const statusMarkW = 2

// statusInset is the indent every [S] row carries — the headers sit at 1 and
// their rows at 3, the depth every modal in the app uses (HUM LEAD, UAT
// 2026-08-28). It rides in the first column because the kit lays out from
// column zero and a table that starts where the header does reads as a
// continuation of it.
//
// IT CANNOT DRIFT FROM modalInset, and it stays a CONST to do it. This comment
// and columns.go's both cited the same HUM LEAD ruling while carrying their own
// copy of the number (D-1, red team 2026-09-05) — but deriving it with
// strings.Repeat would make it a package-level variable, which P10-06 refuses
// and rightly: it would then be writable. The assertion below fails to COMPILE
// if the two ever disagree, which is the stronger guarantee anyway.
const statusInset = "   "

// The compile-time tie. A negative or over-long index here is a build error, so
// statusInset can only ever be modalInset cells wide.
var _ = [1]struct{}{}[len(statusInset)-modalInset]

// statusEndpointMax caps how much of the fill the longest host may claim as its
// floor: past this it is truncatable like any other cell, so one very long name
// cannot push the counters off a narrow window.
const statusEndpointMax = 34

// counterCols are the request columns, all right-aligned and fixed: a count
// belongs under its own heading.
func counterCols(names ...string) []render.StatusColumn {
	out := make([]render.StatusColumn, 0, len(names))
	for _, n := range names {
		w := 5
		if n == "TRIES" || n == "BYTES" {
			w = 6
		}
		out = append(out, render.StatusColumn{Header: n, Width: w, Right: true})
	}
	return out
}

// counters are the row's request cells for this form, blank when httpx has not
// counted this host yet — a zero it never measured would read as "no traffic".
func (r endpointRow) counters(form int) []string {
	if !r.seen || form > 1 {
		switch form {
		case 0:
			return []string{"", "", "", "", ""}
		case 1:
			return []string{"", "", ""}
		}
		return nil
	}
	n := func(v int64) string { return strconv.FormatInt(v, 10) }
	if form == 1 {
		return []string{n(r.tries), n(r.net), n(r.cache)}
	}
	return []string{n(r.tries), n(r.net), n(r.cache), n(r.neg), render.HumanBytes(r.bytes)}
}

// endpointRow is one host: who uses it, how it is doing, and its counters.
type endpointRow struct {
	endpoint                      string
	providers                     string
	status                        string // the snapshot's status word, for the glyph
	state                         string // "REF OK" — the role and the status, as the mock reads them
	fetchedAt                     time.Time
	seen                          bool // whether httpx has counters for this host yet
	claimed                       bool // whether a snapshot provider reports this host's health
	age                           time.Duration
	tries, net, cache, neg, bytes int64
}

// endpointRows folds the provider statuses onto their hosts and joins the
// request counters. A provider with no endpoint mapped still gets a row, under
// its own id, rather than vanishing from the diagnostic.
func endpointRows(provs []snapshot.ProviderStatus, st Stats) []endpointRow {
	byHost := map[string]*endpointRow{}
	var order []string
	for _, p := range provs {
		hosts := st.Endpoints[p.ID]
		if len(hosts) == 0 {
			hosts = []string{p.ID} // unmapped: name it rather than drop it
		}
		for _, h := range hosts {
			r, ok := byHost[h]
			if !ok {
				r = &endpointRow{endpoint: h, claimed: true}
				byHost[h], order = r, append(order, h)
			}
			if r.providers == "" {
				r.providers = strings.ToUpper(p.ID)
			} else {
				r.providers += " · " + strings.ToUpper(p.ID)
			}
			// The WORST status on a shared host wins, and the OLDEST fetch: a
			// row that says ok because one of its two feeds is fine would hide
			// the one that is not.
			if r.status == "" || worseStatus(p.Status, r.status) {
				r.status, r.state = p.Status, roleState(p)
			}
			if r.fetchedAt.IsZero() || (!p.FetchedAt.IsZero() && p.FetchedAt.Before(r.fetchedAt)) {
				r.fetchedAt = p.FetchedAt
			}
		}
	}
	for _, h := range st.Requests.Hosts {
		r, ok := byHost[h.Host]
		if !ok {
			// A HOST NO PROVIDER CLAIMS still gets a row. The ticker's feeds and
			// the geocoder are counted by httpx and are not snapshot providers,
			// so dropping them would have this window under-report the traffic it
			// exists to account for.
			r = &endpointRow{endpoint: h.Host, providers: "—", state: "not reported"}
			byHost[h.Host], order = r, append(order, h.Host)
		}
		r.seen = true
		r.tries, r.net, r.cache, r.neg, r.bytes = h.Attempts, h.Net, h.Cache, h.Neg, h.BytesNet
	}
	out := make([]endpointRow, 0, len(order))
	for _, h := range order {
		out = append(out, *byHost[h])
	}
	return out
}

// roleState is the mock's STATUS cell: the role and the status in one word
// pair — "REF OK", "2ND DOWN".
func roleState(p snapshot.ProviderStatus) string {
	role := "2ND"
	if strings.HasPrefix(p.Role, "ref") {
		role = "REF"
	}
	return role + " " + strings.ToUpper(p.Status)
}

// worseStatus reports whether a is worse than b, so a shared host shows the
// unhealthiest of its feeds.
func worseStatus(a, b string) bool { return statusRank(a) > statusRank(b) }

func statusRank(s string) int {
	switch s {
	case snapshot.ProviderOK:
		return 0
	case snapshot.ProviderDegraded:
		return 2
	}
	return 1 // off, or anything unrecognised
}

// pipelineLines is the PIPELINES table (the mock's columns).
func (d Dashboard) pipelineLines(o render.Opts, st Stats, fillTo int) []string {
	text := render.Tok(render.TextBase)
	// A LADDER, like the providers table: at 80 columns the full set is a cell
	// wider than the window and the clamp ate the S off ROWS. The counters go in
	// the order they are least missed — FOLDED, then PUBLISHES, then LOCATIONS —
	// so what survives at the narrowest width is what a pipeline is judged by:
	// its name, when it last ran, and how much it is holding.
	drop := []string{"FOLDED", "PUBLISHES", "LOCATIONS"}
	cells := [][]string{
		pipeRow("PRIORITY", o, d.snap, st.Pipelines[0], ""),
		pipeRow("RECENT", o, d.recent, st.Pipelines[1], ""),
		{"SEVERE INDEX", "-", "-", "-", "-", fmt.Sprintf("%d/%d", len(d.severe.Rows), SevereMaxRows)},
	}
	avail := statusAvail(o)
	var cols []render.StatusColumn
	var keep []int
	for form := 0; form <= len(drop); form++ { // bounded by the drop list (P10-02)
		cols, keep = pipelineCols(drop[:form])
		if render.StatusNaturalWidth(cols) <= avail {
			break
		}
	}
	rows := make([]render.StatusRow, 0, len(cells))
	for _, c := range cells {
		kept := make([]string, 0, len(keep))
		for _, i := range keep {
			kept = append(kept, c[i])
		}
		styles := map[int]string{}
		for i := range kept {
			styles[i] = text
		}
		kept[0] = statusInset + kept[0]
		rows = append(rows, render.StatusRow{Cells: kept, Styles: styles})
	}
	out := []string{statusHeader("PIPELINES")}
	return append(out, o.StatusTable(cols, rows, statusFill(cols, fillTo), render.Tok(render.ModalTitle))...)
}

// pipelineCols is the PIPELINES column set less the named columns.
// It returns the columns and the indexes of the cells they take, so a row is
// built from the same list whichever form is in play.
func pipelineCols(without []string) ([]render.StatusColumn, []int) {
	all := []render.StatusColumn{
		{Header: statusInset + "NAME", Fill: true, MinWidth: len(statusInset) + len("SEVERE INDEX")},
		{Header: "SNAPSHOT", Width: 11, Right: true},
		{Header: "LOCATIONS", Width: 9, Right: true},
		{Header: "PUBLISHES", Width: 9, Right: true},
		{Header: "FOLDED", Width: 6, Right: true},
		{Header: "ROWS", Width: 7, Right: true},
	}
	cols, keep := make([]render.StatusColumn, 0, len(all)), make([]int, 0, len(all))
	for i, c := range all {
		if slices.Contains(without, strings.TrimSpace(c.Header)) {
			continue
		}
		cols, keep = append(cols, c), append(keep, i)
	}
	return cols, keep
}

// pipeRow is one pipeline's cells, in the full column order.
func pipeRow(name string, o render.Opts, sn *snapshot.Snapshot, ps PipelineStats, rows string) []string {
	stamp, locs := "-", "-"
	if sn != nil {
		stamp = o.Clock.TimeSec(dataAsOf(sn).Local())
		locs = strconv.Itoa(len(sn.Locations))
	}
	if rows == "" {
		rows = "-"
	}
	return []string{name, stamp, locs, strconv.FormatInt(ps.Publishes, 10), strconv.FormatInt(ps.Folded, 10), rows}
}

// statusFill is the width to lay a table out at: the window's inner width once
// it is known, and the table's own natural width on the measuring pass that
// decides what that width will be.
func statusFill(cols []render.StatusColumn, fillTo int) int {
	if fillTo > 0 {
		return fillTo
	}
	return render.StatusNaturalWidth(cols)
}

// dumpLines is the [S] DUMPS block: the diagnostic dump's last outcome and
// the trigger for this platform.
func dumpLines(st Stats) []string {
	lines := []string{statusHeader("DUMPS")}
	if st.LastDump == "" {
		lines = append(lines, "   none yet")
	} else {
		lines = append(lines, "   last: "+st.LastDump)
	}
	if st.DumpHint != "" {
		lines = append(lines, "   trigger: "+st.DumpHint)
	}
	return lines
}

// fixedAge formats a duration in a 7-cell right-aligned slot so ages line
// up: "59m 59s", " 1m  5s", "    55s", " 2h 05m".
func fixedAge(dur time.Duration) string {
	dur = dur.Round(time.Second)
	h, m, sec := int(dur.Hours()), int(dur.Minutes())%60, int(dur.Seconds())%60
	switch {
	case h > 0:
		return fmt.Sprintf("%2dh %02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%2dm %2ds", m, sec)
	}
	return fmt.Sprintf("    %2ds", sec)
}

// issue is one aggregated warning class.
type issue struct {
	code, provider, latest string
	count, locations       int
	endpoint, blame        string // the structured half (snapshot.WithFailure)
	status                 int    // the HTTP status, 0 when the failure had none
}

// issueLines is the ISSUES table: what failed, where, with what, and WHOSE
// FAULT it looks like.
//
// The blame column is the point of the table. A 5xx is theirs, a request we
// built wrong is ours, and the two that turn on a credential — 401 and 403 —
// read "unclear" rather than guessing, because a diagnostic that blames the
// wrong side sends somebody to fix the wrong thing (snapshot.BlameFor).
func (d Dashboard) issueLines(o render.Opts, fillTo int, snaps ...*snapshot.Snapshot) []string {
	issues := foldWarnings(snaps)
	if len(issues) == 0 {
		return []string{"   none"}
	}
	shown := issues
	trimmed := ""
	if len(shown) > maxIssueRows {
		shown = shown[:maxIssueRows]
		trimmed = fmt.Sprintf("   ... and %d more issue classes", len(issues)-maxIssueRows)
	}
	// The same ladder the providers and pipelines tables use: on a narrow window
	// a column is dropped WHOLE rather than clipped, because a heading with a
	// cut-off cell under it claims a number the reader cannot actually see.
	// BLAME goes first — the message line beneath each row states the cause in
	// words — then SEEN.
	drop := []string{"BLAME", "SEEN"}
	var cols []render.StatusColumn
	var keep []int
	avail := statusAvail(o)
	for form := 0; form <= len(drop); form++ {
		cols, keep = issueCols(shown, drop[:form])
		// An unknown terminal width drops nothing: a caller with no Opts is
		// asking what the table HOLDS, not how it fits.
		if avail <= 0 || render.StatusNaturalWidth(cols) <= avail {
			break
		}
	}
	rows, tails := issueRows(o, shown, keep)
	laid := o.StatusTable(cols, rows, statusFill(cols, fillTo), render.Tok(render.ModalTitle))
	// THE MESSAGE IS NOT A COLUMN — it is interleaved under its own row, so the
	// table stays a table and the sentence stays a sentence.
	//
	// WRAPPED HERE, not left to the panel. The panel wraps to its own content
	// width, which is wider than the tables by the scroll rail — so a message
	// left to it overran the columns above it by exactly the rail. Wrapping at
	// the table's width keeps the block a rectangle.
	tailInset := "       "
	tailW := max(20, widest(laid)-len(tailInset))
	out := []string{laid[0]}
	for i, line := range laid[1:] {
		out = append(out, line)
		if tails[i] == "" {
			continue
		}
		for _, w := range render.WrapText(tails[i], tailW) {
			out = append(out, tailInset+w)
		}
	}
	if trimmed != "" {
		out = append(out, trimmed)
	}
	return out
}

// issueCols sizes the ISSUES columns from what is in them.
//
// ENDPOINT FILLS, as it does on the providers table: it is the cell that varies
// most and the one a wide terminal has room to show in full.
func issueCols(issues []*issue, without []string) ([]render.StatusColumn, []int) {
	provW, epW, errW := len("PROVIDER"), len("ENDPOINT"), len("ERROR")
	blameW, seenW := len("BLAME"), len("SEEN")
	for _, it := range issues {
		provW, epW = max(provW, render.Width(it.provider)), max(epW, render.Width(it.endpoint))
		errW = max(errW, render.Width(errorWord(it)))
		blameW, seenW = max(blameW, render.Width(it.blame)), max(seenW, render.Width(issueScope(it)))
	}
	// SEEN is not in the mock, and it is kept because the table FOLDS: without
	// it, three stale observations across two locations and one across one read
	// exactly alike (HUM LEAD's mock, UAT 2026-08-30 — flagged).
	all := []render.StatusColumn{
		{Header: statusInset + strings.Repeat(" ", statusMarkW) + "PROVIDER",
			Width: len(statusInset) + statusMarkW + provW},
		{Header: "ENDPOINT", Fill: true, MinWidth: epW, Truncatable: true},
		{Header: "ERROR", Width: errW},
		{Header: "BLAME", Width: blameW},
		{Header: "SEEN", Width: seenW},
	}
	cols, keep := make([]render.StatusColumn, 0, len(all)), make([]int, 0, len(all))
	for i, c := range all {
		if slices.Contains(without, strings.TrimSpace(c.Header)) {
			continue
		}
		cols, keep = append(cols, c), append(keep, i)
	}
	return cols, keep
}

// issueRows is one row per issue class, and the message that belongs under it.
func issueRows(o render.Opts, issues []*issue, keep []int) ([]render.StatusRow, []string) {
	text := render.Tok(render.TextBase)
	rows, tails := make([]render.StatusRow, 0, len(issues)), make([]string, 0, len(issues))
	for _, it := range issues {
		glyph, tone := o.Glyphs().Alert, render.Tok(render.AlertLabel)
		if it.code == snapshot.WarnProviderError {
			glyph, tone = o.Glyphs().Fail, render.Tok(render.ProviderDown)
		}
		all := []string{
			statusInset + render.PadTo(glyph, statusMarkW) + strings.ToUpper(dashIfEmpty(it.provider)),
			dashIfEmpty(it.endpoint), errorWord(it), dashIfEmpty(it.blame), issueScope(it),
		}
		cells := make([]string, 0, len(keep))
		for _, i := range keep {
			cells = append(cells, all[i])
		}
		styles := map[int]string{}
		for j := range cells {
			styles[j] = text
		}
		styles[0] = tone // the provider wears the severity, as the endpoint does above
		rows = append(rows, render.StatusRow{Cells: cells, Styles: styles})
		tails = append(tails, it.latest)
	}
	return rows, tails
}

// maxIssueRows bounds the table: warnings are provider-supplied and a bad day
// can produce a page of them.
const maxIssueRows = 8

// errorWord is the ERROR cell: the HTTP status when there was one, else the
// warning's own code — "HTTP 502", or "obs_stale" for something that never
// touched the network.
func errorWord(it *issue) string {
	if it.status > 0 {
		return fmt.Sprintf("HTTP %d", it.status)
	}
	return it.code
}

// issueScope is how many times this class was seen, and across how many
// locations when it is a per-location warning.
func issueScope(it *issue) string {
	s := fmt.Sprintf("×%d", it.count)
	if it.locations > 0 {
		s += fmt.Sprintf(" (%d loc)", it.locations)
	}
	return s
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// foldWarnings groups by code+provider: count, distinct locations, latest
// message; provider errors sort first, then by count.
func foldWarnings(snaps []*snapshot.Snapshot) []*issue {
	byKey := map[string]*issue{}
	seenLoc := map[string]map[string]bool{}
	var order []string // first-seen order: a map's range is deliberately random
	for _, sn := range snaps {
		if sn == nil {
			continue
		}
		for _, w := range sn.Warnings {
			k := w.Code + "|" + w.Provider
			it := byKey[k]
			if it == nil {
				it = &issue{code: w.Code, provider: w.Provider}
				byKey[k] = it
				order = append(order, k)
				seenLoc[k] = map[string]bool{}
			}
			it.count++
			// THE ROW DESCRIBES ONE OCCURRENCE — the latest. Keeping the first
			// occurrence's endpoint, status and blame beside the last one's
			// message produced a row that read "HTTP 502 · provider_error" over
			// a sentence about a 404, and the blame column is the point of this
			// table: a diagnostic that names the wrong side sends somebody to
			// fix the wrong thing.
			it.latest, it.endpoint, it.blame, it.status = w.Message, w.Endpoint, w.Blame, w.HTTPStatus
			if w.Location != "" && !seenLoc[k][w.Location] {
				seenLoc[k][w.Location] = true
				it.locations++
			}
		}
	}
	issues := make([]*issue, 0, len(byKey))
	for _, k := range order {
		issues = append(issues, byKey[k])
	}
	// STABLE, AND TOTALLY ORDERED. Collecting from a map and sorting on a
	// comparator that ties left the row order to Go's random map iteration:
	// identical input produced a different table on every render, and the
	// maxIssueRows cut then hid a different class each time. The final tiebreak
	// makes the order a function of the data alone.
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if fatal := a.code == snapshot.WarnProviderError; fatal != (b.code == snapshot.WarnProviderError) {
			return fatal
		}
		if a.count != b.count {
			return a.count > b.count
		}
		return a.code+"|"+a.provider < b.code+"|"+b.provider
	})
	return issues
}

// providersOf guards the nil snapshot for the status view.
func providersOf(sn *snapshot.Snapshot) []snapshot.ProviderStatus {
	if sn == nil {
		return nil
	}
	return sn.Providers
}
