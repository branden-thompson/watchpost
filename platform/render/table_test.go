package render

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestHeaderConstsMatchMockFile(t *testing.T) {
	// The mock IS the spec (feedback-mock-fidelity): header constants must
	// match the committed mock file character-for-character.
	raw, err := os.ReadFile("../../06_docs/02_features/watchpost-cli/09-view-mocks/watchpost-cli-view-mocks-with-notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	wantGroup := strings.TrimRight(lines[26][:len(mockGroupHeader)], " ")
	wantCols := strings.TrimRight(lines[27][:len(mockColHeader)], " ")
	if mockGroupHeader != wantGroup {
		t.Fatalf("group header drifted from mock:\n got %q\nwant %q", mockGroupHeader, wantGroup)
	}
	if mockColHeader != wantCols {
		t.Fatalf("column header drifted from mock:\n got %q\nwant %q", mockColHeader, wantCols)
	}
}

func TestHeaderTokensAtResolvedOffsets(t *testing.T) {
	// LABEL hidden (UAT 11.2) and NAME is the fill column (UAT 11.1): at the
	// minimal full width (115) every downstream token sits at the computed
	// offset; widening moves them right by exactly the fill growth.
	hdr := stripANSI(strings.Split((Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable(nil, 0), "\n")[1])
	for tok, off := range map[string]int{"###.": 13, "NAME": 18, "ZIP": 37, "NOW": 60} {
		if got := runeIdx(hdr, tok); got != off {
			t.Fatalf("%q at %d, want %d\n%q", tok, got, off, hdr)
		}
	}
	if strings.Contains(hdr, "LABEL") {
		t.Fatalf("LABEL must be hidden (UAT 11.2): %q", hdr)
	}
	wide := stripANSI(strings.Split((Opts{ThinBands: true, Width: 135, Units: UnitF}).LocationTable(nil, 0), "\n")[1])
	if got := runeIdx(wide, "ZIP"); got != 57 {
		t.Fatalf("fill must widen NAME by +20 at 135 cols: ZIP at %d, want 57\n%q", got, wide)
	}
}

func TestLocationTableAnatomy(t *testing.T) {
	out := (Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{testRow()}, 0)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expect group+header+1 row, got %d:\n%s", len(lines), out)
	}
	row := []rune(stripANSI(lines[2]))
	// Absolute offsets of the label-less minimal-full layout; temps
	// right-justify in 5-cell fields (mock rows " 72ºF"/" 00ºF").
	for _, c := range []struct {
		col  int
		want string
	}{
		{0, "›"}, {3, "▶"}, {11, "⚠"},
		{13, "001."}, {18, "Oceanside, CA"}, {37, "92057"},
		{46, "CLEAR"}, {60, " 73ºF↗"}, {70, " 75ºF"}, {77, " 63ºF"},
		{86, "RAIN"}, {100, " 77ºF"}, {108, " 64ºF"},
	} {
		got := string(row[c.col:min(len(row), c.col+len([]rune(c.want)))])
		if got != c.want {
			t.Fatalf("col %d = %q, want %q\n%s", c.col, got, c.want, out)
		}
	}
	if strings.Contains(stripANSI(out), "OSIDE") {
		t.Fatalf("LABEL data must not render (UAT 11.2):\n%s", out)
	}
	for _, l := range lines {
		if w := displayWidth(l); w > 115 {
			t.Fatalf("line exceeds width %d: %q", w, l)
		}
	}
}

func TestResponsiveColumnDropsInOrder(t *testing.T) {
	// UAT-2D order with LABEL hidden: TOMORROW leaves first, then TODAY
	// HI/LOW; the NAME fill keeps every layout flush to the width.
	r := testRow()
	at := func(w int) string {
		return stripANSI((Opts{ThinBands: true, Width: w, Units: UnitF}).LocationTable([]LocationRow{r}, 0))
	}

	full := at(121)
	if !strings.Contains(full, "T O M O R R O W") || !strings.Contains(full, "LOW") {
		t.Fatalf("121 cols must carry all groups:\n%s", full)
	}
	if got := runeIdx(strings.Split(full, "\n")[1], "ZIP"); got != 43 {
		t.Fatalf("fill must widen NAME (+6 at 121 cols): ZIP at %d, want 43\n%s", got, full)
	}
	noTmrw := at(100)
	if strings.Contains(noTmrw, "T O M O R R O W") || strings.Contains(noTmrw, "RAIN") {
		t.Fatalf("<115 cols must drop the TOMORROW group:\n%s", noTmrw)
	}
	if !strings.Contains(noTmrw, "LOW") {
		t.Fatalf("TODAY HI/LOW must survive the first drop:\n%s", noTmrw)
	}
	minimal := at(80)
	if strings.Contains(minimal, "LOW") || strings.Contains(minimal, " 75ºF") {
		t.Fatalf("<84 cols must drop TODAY HI/LOW:\n%s", minimal)
	}
	for _, out := range []string{full, noTmrw, minimal} {
		if !strings.Contains(out, "Oceanside, CA") || !strings.Contains(out, "92057") || !strings.Contains(out, " 73ºF↗") {
			t.Fatalf("core columns (NAME/ZIP/NOW) must survive every breakpoint:\n%s", out)
		}
	}
}

func TestExtendedColumnsBeyond125(t *testing.T) {
	// Ultra-wide: EXTENDED day columns claim width beyond the minimal full
	// layout; the NAME fill absorbs the remainder so rows stay flush.
	r := testRow()
	r.Extended = []DayCell{
		{Date: "08/26", Hi: f64(30.0), Lo: f64(20.0)},
		{Date: "08/27", Hi: f64(31.0), Lo: f64(21.0)},
		{Date: "08/28", Hi: f64(32.0), Lo: f64(22.0)},
		{Date: "08/29", Hi: f64(33.0), Lo: f64(23.0)},
		{Date: "08/30", Hi: f64(34.0), Lo: f64(24.0)},
	}
	// 240 (was 220): WX STN + DIST take the first 16 cells beyond the minimal
	// full layout (UAT 60); the five day columns claim the width beyond that.
	out := stripANSI((Opts{ThinBands: true, Width: 240, Units: UnitF}).LocationTable([]LocationRow{r}, 5))
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[0], "E X T E N D E D   F O R E C A S T") {
		t.Fatalf("ultra-wide group header missing:\n%s", lines[0])
	}
	if n := strings.Count(lines[2], "H "); n != 5 {
		t.Fatalf("want 5 extended day cells, got %d:\n%s", n, lines[2])
	}
	// mm/dd header sits 5 cells into its day column.
	if h, c := runeIdx(lines[1], "08/26"), runeIdx(lines[2], "H "); h != c+5 {
		t.Fatalf("date header must align with its cell: hdr %d cell %d", h, c)
	}
	if !strings.Contains(lines[2], "H  86ºF/68ºF  L") {
		t.Fatalf("fixed-slot day cell missing:\n%s", lines[2])
	}
	// Narrower ultra-wide terminals get fewer day columns, never a torn cell.
	narrow := stripANSI((Opts{ThinBands: true, Width: 150, Units: UnitF}).LocationTable([]LocationRow{r}, 5))
	if n := strings.Count(strings.Split(narrow, "\n")[2], "H "); n != 1 {
		t.Fatalf("150 cols fits exactly one extended day, got %d:\n%s", n, narrow)
	}
	if got := (Opts{ThinBands: true, Width: 240}).TableRowLen(5); got != 240 {
		t.Fatalf("TableRowLen must span the width under fill, got %d", got)
	}
}

func TestSessionFourStyling(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	// 4.8: provider health green when OK, red otherwise.
	o := Opts{ThinBands: true, Width: 125, Units: UnitF}
	if g := o.HealthGlyph("NWS", snapshot.ProviderOK); !strings.Contains(g, "38;5;77") {
		t.Fatalf("healthy provider must be green: %q", g)
	}
	if g := o.HealthGlyph("NWS", snapshot.ProviderDegraded); !strings.Contains(g, "38;5;196") {
		t.Fatalf("degraded provider must be red: %q", g)
	}
	// 4.7: focused row's name bold yellow; 4.5: n/a temps in base grey.
	r := testRow()
	r.Hi = nil
	out := o.LocationTable([]LocationRow{r}, 0)
	rowLine := strings.Split(out, "\n")[2]
	if !strings.Contains(rowLine, "1;38;5;220") {
		t.Fatalf("focused name must be bold yellow:\n%q", rowLine)
	}
	// UAT 50 supersedes the focused-row case: n/a on the FOCUSED row reads
	// the light-blue focus tone; unfocused rows keep the base grey.
	if !strings.Contains(rowLine, "38;5;117") {
		t.Fatalf("n/a HI on the focused row must read the focus tone:\n%q", rowLine)
	}
	r.Selected = false
	if un := strings.Split(o.LocationTable([]LocationRow{r}, 0), "\n")[2]; !strings.Contains(un, "38;5;250") {
		t.Fatalf("n/a HI on an unfocused row must fall back to base grey:\n%q", un)
	}
	// 4.6: alert tone — warning-grade red, watch-grade yellow.
	if AlertTone("Extreme Heat Watch", "moderate") != Tok(AlertLabel) {
		t.Fatal("watch-grade must read yellow")
	}
	if AlertTone("Flash Flood Warning", "moderate") != Tok(AlertDanger) || AlertTone("Heat Advisory", "severe") != Tok(AlertDanger) {
		t.Fatal("warning-grade / severe must read red")
	}
	if p := o.PanelColored("T", "x", Tok(AlertDanger)); !strings.Contains(p, "38;5;196") {
		t.Fatalf("panel border tint missing: %q", p)
	}
	// 4.9: gradient title — bold truecolor, the reference stops at the ends.
	g := TitleGradient("W A T C H P O S T")
	if !strings.Contains(g, "1;38;2;221;81;214") || !strings.Contains(g, "38;2;124;227;179") {
		t.Fatalf("gradient stops missing: %q", g)
	}
	// 4.10: base tint re-arms after every reset and closes clean.
	tinted := TintDefault("plain \x1b[0m tail")
	if !strings.HasPrefix(tinted, "\x1b[0;38;5;250m") || !strings.Contains(tinted, "\x1b[0m tail") == strings.Contains(tinted, " tail") == false {
		t.Fatalf("tint: %q", tinted)
	}
	if !strings.HasSuffix(tinted, "\x1b[0m") {
		t.Fatalf("tint must close clean: %q", tinted)
	}
	// 4.3: group chips cover the column that extends past the mock bracket.
	full := o.LocationTable([]LocationRow{testRow()}, 0)
	head := strings.Split(full, "\n")[0]
	if strings.Contains(stripANSI(head), "]  ") && strings.Contains(head, "[") {
		t.Fatalf("group chips must swallow brackets when styled: %q", head)
	}
}

func TestAlertNameAndTrendTones(t *testing.T) {
	// UAT 14.1/14.2: alerted names muted yellow (advisory) / muted red
	// (warning); trend arrows muted orange up / muted cyan down.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	o := Opts{ThinBands: true, Width: 125, Units: UnitF}
	adv := testRow()
	adv.Selected, adv.HasAlert, adv.WarnAlert = false, true, false
	warn := testRow()
	warn.Selected, warn.HasAlert, warn.WarnAlert = false, true, true
	out := o.LocationTable([]LocationRow{adv, warn}, 0)
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[2], "38;5;186") {
		t.Fatalf("advisory row name must be muted yellow:\n%q", lines[2])
	}
	if !strings.Contains(lines[3], "38;5;174") {
		t.Fatalf("warning row name must be muted red:\n%q", lines[3])
	}
	if !strings.Contains(lines[2], "38;5;137m↗") {
		t.Fatalf("up-trend arrow must be muted orange:\n%q", lines[2])
	}
	if g := (Opts{}).TrendGlyph("down"); !strings.Contains(g, "38;5;73m↘") {
		t.Fatalf("down-trend arrow must be muted cyan: %q", g)
	}
	if !AlertIsWarning("Flash Flood Warning", "moderate") || AlertIsWarning("Heat Advisory", "minor") {
		t.Fatal("AlertIsWarning classification")
	}
}

func TestGroupBandsTouchAtGutterMidpoints(t *testing.T) {
	// UAT 14.4: neighboring bands meet - no gaps in the strip (color on);
	// color off keeps the bracketed form.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	out := (Opts{ThinBands: true, Width: 125, Units: UnitF}).LocationTable([]LocationRow{testRow()}, 0)
	raw := strings.Split(out, "\n")[0]
	// Adjacent bands close and immediately reopen - any plain spaces between
	// a band's reset and the next band's escape is a visual gap.
	if regexp.MustCompile(`\x1b\[0m +\x1b\[`).MatchString(raw) {
		t.Fatalf("bands must touch (no unstyled gap between spans): %q", raw)
	}
	if w := displayWidth(raw); w != 125 {
		t.Fatalf("band strip must span the table width, got %d", w)
	}
}

func TestExtendedBandShortLabelWhenNarrow(t *testing.T) {
	// UAT 16.1: a one-day EXTENDED band degrades to "E X T E N D E D",
	// never a truncated "EXTENDEDFORECA…".
	r := testRow()
	r.Extended = []DayCell{{Date: "08/26", Hi: f64(30.0), Lo: f64(20.0)}}
	out := stripANSI((Opts{ThinBands: true, Width: 150, Units: UnitF}).LocationTable([]LocationRow{r}, 1))
	band := strings.Split(out, "\n")[0]
	if strings.Contains(band, "…") || strings.Contains(band, "EXTENDEDF") {
		t.Fatalf("narrow extended band must use the short title: %q", band)
	}
	if !strings.Contains(band, "E X T E N D E D") {
		t.Fatalf("short title missing: %q", band)
	}
}

func TestAlertBadgeCountAndTone(t *testing.T) {
	// UAT 20: header note icon hidden (column kept); '›  2⚠' badge beside
	// the glyph, toned by the most severe alert (yellow advisory, red
	// warning); go-studs spacing untouched.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	hdr := stripANSI(strings.Split((Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable(nil, 0), "\n")[1])
	if strings.Contains(hdr, "♪") {
		t.Fatalf("header note icon must be hidden (UAT 20.1): %q", hdr)
	}
	warn := testRow()
	warn.HasAlert, warn.WarnAlert, warn.AlertCount, warn.Playing = true, true, 2, false
	adv := testRow()
	adv.Selected, adv.HasAlert, adv.WarnAlert, adv.AlertCount, adv.Playing = false, true, false, 1, false
	out := (Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{warn, adv}, 0)
	lines := strings.Split(out, "\n")
	plain := []rune(stripANSI(lines[2]))
	if got := string(plain[0:13]); got != "›         2⚠ " {
		t.Fatalf("focused 2-alert row must open '›         2⚠ ' (0.11.0 marks block), got %q", got)
	}
	if !strings.Contains(lines[2], "\x1b[38;5;196m2") || !strings.Contains(lines[2], "\x1b[38;5;196m⚠") {
		t.Fatalf("warning badge must be red:\n%q", lines[2])
	}
	if !strings.Contains(lines[3], "\x1b[38;5;220m1") || !strings.Contains(lines[3], "\x1b[38;5;220m⚠") {
		t.Fatalf("advisory badge must be yellow:\n%q", lines[3])
	}
	// Geometry untouched: NAME still lands at col 18 (0.11.0 marks block).
	if got := runeIdx(stripANSI(lines[2]), "Oceanside"); got != 18 {
		t.Fatalf("badge must not disturb column geometry: NAME at %d", got)
	}
}

func TestFocusedRowCellsLightBlueAndPointerBoldWhite(t *testing.T) {
	// UAT 50: on the focused row the grey data cells read light blue (name
	// and HI/LO temp tones unchanged) and the pointer is bold white.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	row := strings.Split((Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{testRow()}, 0), "\n")[2]
	if !strings.Contains(row, "\x1b[1;97m›") {
		t.Fatalf("pointer must be bold white:\n%q", row)
	}
	if !strings.Contains(row, "\x1b[38;5;117m92057") || !strings.Contains(row, "\x1b[38;5;117m001.") {
		t.Fatalf("focused zip/num cells must be light blue:\n%q", row)
	}
	if !strings.Contains(row, "38;5;208") || !strings.Contains(row, "1;38;5;220") {
		t.Fatalf("temp tones and the name tone must be unchanged:\n%q", row)
	}
	unfocused := testRow()
	unfocused.Selected = false
	if r := strings.Split((Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{unfocused}, 0), "\n")[2]; strings.Contains(r, "38;5;117") {
		t.Fatal("unfocused rows must not use the focus cell tone")
	}
}

func TestStationColumnsUAT60(t *testing.T) {
	// UAT 60: WX STN (observing station) and DIST (its distance) sit between
	// NAME and ZIP from 131 cols; they leave before ZIP as the width narrows;
	// ZIP is the last identity column to go (only when NAME would drop below
	// its 10-cell floor). "WX" keeps the header apart from the NOAA radio
	// transmitter callsign the player will carry.
	r := testRow()
	r.Station, r.StationKM = "KCRQ", f64(5.5)
	at := func(w int, u Units) []string {
		return strings.Split(stripANSI((Opts{ThinBands: true, Width: w, Units: u}).LocationTable([]LocationRow{r}, 0)), "\n")
	}
	full := at(131, UnitF)
	for tok, off := range map[string]int{"NAME": 18, "WX STN": 37, "DIST": 45, "ZIP": 53, "CONDITIONS": 62} {
		if got := runeIdx(full[1], tok); got != off {
			t.Fatalf("%q at %d, want %d\n%q", tok, got, off, full[1])
		}
	}
	row := []rune(full[2])
	if got := string(row[37:41]); got != "KCRQ" {
		t.Fatalf("station cell = %q\n%s", got, full[2])
	}
	if got := string(row[45:51]); got != "  3 mi" {
		t.Fatalf("distance renders in the display units (miles under ºF): %q", got)
	}
	if got := string([]rune(at(131, UnitC)[2])[45:51]); got != "  6 km" {
		t.Fatalf("distance under ºC: %q", got)
	}
	band := full[0]
	if i, j := runeIdx(band, "L O C A T I O N"), runeIdx(band, "T O D A Y"); i < 0 || j < 62-2 {
		t.Fatalf("LOCATION band must widen over the new columns:\n%s", band)
	}
	// Loading / unknown: cells stay blank rather than lying.
	blank := testRow()
	if out := stripANSI((Opts{ThinBands: true, Width: 131, Units: UnitF}).LocationTable([]LocationRow{blank}, 0)); strings.Contains(out, " mi") {
		t.Fatalf("unknown distance must render blank:\n%s", out)
	}
	// Below 131 the station columns leave first; TOMORROW survives at 125.
	mock := at(125, UnitF)
	if strings.Contains(mock[1], "WX STN") || strings.Contains(mock[2], "KCRQ") || !strings.Contains(mock[0], "T O M O R R O W") {
		t.Fatalf("125 cols: station columns hidden, TOMORROW kept:\n%s", strings.Join(mock, "\n"))
	}
	if got := runeIdx(mock[1], "ZIP"); got != 47 {
		t.Fatalf("125-col layout unchanged (ZIP at 47), got %d", got)
	}
	// Very narrow: ZIP leaves last, NAME keeps its floor. The 0.11.0 marks
	// block (13 cells) sets the table's minimum content width at 52 (the band
	// labels span the marks region); the full name still survives from 55.
	narrow := at(52, UnitF)
	if strings.Contains(narrow[1], "ZIP") || strings.Contains(narrow[2], "92057") {
		t.Fatalf("52 cols must drop ZIP before breaking NAME:\n%s", strings.Join(narrow, "\n"))
	}
	if !strings.Contains(narrow[2], "Oceansi") || !strings.Contains(narrow[2], " 73ºF") {
		t.Fatalf("NAME and NOW survive every width:\n%s", strings.Join(narrow, "\n"))
	}
	if n55 := at(55, UnitF); !strings.Contains(n55[2], "Oceanside") {
		t.Fatalf("55 cols carries the full name:\n%s", strings.Join(n55, "\n"))
	}
	for _, l := range narrow {
		if w := displayWidth(l); w > 52 {
			t.Fatalf("line exceeds 52: %q", l)
		}
	}
}

func TestFireMarkIsACountedOrangeDiamond(t *testing.T) {
	// 0.11.0 mock `›  ▶ ● 5◆ 3⚠ 009.`: fire is a counted orange ◆ in its own
	// slot pair (bold when a hotspot burns hard), * under --ascii, never
	// displacing the play, seismic or alert marks; NAME's floor gives up the
	// cells. Here the seismic slot is empty (no quake), so ▶ and the fire
	// count are three cells apart.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	r := testRow()
	r.Fire, r.FireHot, r.AlertCount = 3, true, 2
	out := (Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0)
	plain := []rune(stripANSI(strings.Split(out, "\n")[2]))
	if got := string(plain[0:13]); got != "›  ▶   3◆ 2⚠ " {
		t.Fatalf("marks block: %q", got)
	}
	if bold := Tint("◆", "1;"+Tok(FireMark)); !strings.Contains(out, bold) {
		t.Fatalf("a hot row wears a bold ◆:\n%q", out)
	}
	r.FireHot = false
	if out := (Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0); strings.Contains(out, Tint("◆", "1;"+Tok(FireMark))) {
		t.Fatal("a cool row must not read bold")
	}
	ascii := stripANSI((Opts{ThinBands: true, Width: 115, Units: UnitF, ASCII: true}).LocationTable([]LocationRow{r}, 0))
	if !strings.Contains(ascii, "3*") || strings.Contains(ascii, "◆") {
		t.Fatalf("--ascii swaps ◆ for *:\n%s", ascii)
	}
	r.Fire = 0
	if strings.Contains(stripANSI((Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0)), "◆") {
		t.Fatal("no fire, no mark")
	}
}

// 0.11.0 (HUM LEAD): the strongest recent quake's felt-band glyph sits in the
// marks block between the play and fire marks — one glyph, no count, violet.
func TestSeismicMarkIsStrongestQuakeGlyph(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	// The ramp ○●◉ at cell 5, in the SeismicMark tone, without disturbing the
	// fire/alert marks or the downstream columns.
	for level, glyph := range map[int]string{1: "○", 2: "●", 3: "◉"} {
		r := testRow()
		r.Seismic, r.Fire, r.AlertCount = level, 3, 2
		out := (Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0)
		plain := []rune(stripANSI(strings.Split(out, "\n")[2]))
		if got := string(plain[5]); got != glyph {
			t.Fatalf("level %d: seismic glyph at cell 5 = %q, want %q", level, got, glyph)
		}
		if !strings.Contains(out, Tint(glyph, Tok(SeismicMark))) {
			t.Fatalf("level %d: the seismic mark must read in the violet SeismicMark tone:\n%q", level, out)
		}
		// Fire and alert marks keep their (shifted) slots.
		if got := string(plain[7:9]); got != "3◆" {
			t.Fatalf("fire mark must still render at 7-8, got %q", got)
		}
		if got := runeIdx(stripANSI(strings.Split(out, "\n")[2]), "Oceanside"); got != 18 {
			t.Fatalf("the seismic mark must not disturb NAME@18, got %d", got)
		}
	}
	// ASCII ramp .oO.
	for level, glyph := range map[int]string{1: ".", 2: "o", 3: "O"} {
		r := testRow()
		r.Seismic = level
		ascii := stripANSI(strings.Split((Opts{ThinBands: true, Width: 115, Units: UnitF, ASCII: true}).LocationTable([]LocationRow{r}, 0), "\n")[2])
		if got := string([]rune(ascii)[5]); got != glyph {
			t.Fatalf("--ascii level %d: cell 5 = %q, want %q", level, got, glyph)
		}
	}
	// No quake, no mark.
	r := testRow()
	r.Seismic = 0
	plain := stripANSI(strings.Split((Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0), "\n")[2])
	for _, g := range []string{"○", "●", "◉"} {
		if strings.Contains(plain, g) {
			t.Fatalf("no quake, no seismic mark, found %q", g)
		}
	}
}

// HUM LEAD UAT 2026-08-27: the group bands are three rows — a band-coloured
// blank row above and below the label — unless the layout asked for thin.
func TestGroupBandsAreThreeRowsUnlessThin(t *testing.T) {
	o := Opts{Width: 120} // the anatomy tests above use ThinBands: they test columns, not the band height
	rows := strings.Split(o.LocationTable([]LocationRow{{Index: 1, Name: "X"}}, 0), "\n")
	if len(rows) < 5 || !strings.Contains(rows[1], "L O C A T I O N") || strings.Contains(rows[0], "L") || strings.Contains(rows[2], "L") {
		t.Fatalf("three band rows with the label in the middle:\n%s", strings.Join(rows[:3], "\n"))
	}
	if w0, w1 := displayWidth(rows[0]), displayWidth(rows[1]); w0 != w1 || !strings.HasPrefix(strings.TrimSpace(rows[0]), "[") {
		t.Fatalf("the blank rows span the same cells as the label row (%d vs %d): %q", w0, w1, rows[0])
	}
	o.ThinBands = true
	thin := strings.Split(o.LocationTable([]LocationRow{{Index: 1, Name: "X"}}, 0), "\n")
	if !strings.Contains(thin[0], "L O C A T I O N") || strings.Contains(thin[1], "L O C A T I O N") {
		t.Fatalf("thin: the label row alone:\n%s", strings.Join(thin[:2], "\n"))
	}
	if band := (Opts{}).BandRows("R E C E N T", "R", 40, GroupSectionBG); len(band) != 3 || band[0] != band[2] || !strings.Contains(band[1], "R E C E N T") {
		t.Fatalf("section band rows: %q", band)
	}
}
