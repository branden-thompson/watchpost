package render

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Spec: D-9 seam (ONLY package importing go-studs — table rides go-studs
// DataTable per UAT session 2 ruling), mock M-V1 anatomy at measured absolute
// offsets, UAT-2D responsive breakpoints, D-19 (f/c live units), R-12a
// (severity/health never color-only).

func f64(v float64) *float64 { return &v }

// runeIdx is strings.Index in display columns (the header's ♪ and the rows'
// º are multibyte - byte offsets lie).
func runeIdx(s, tok string) int {
	b := strings.Index(s, tok)
	if b < 0 {
		return -1
	}
	return len([]rune(s[:b]))
}

func TestFormatTempUnits(t *testing.T) {
	if got := (Opts{Units: UnitF}).Temp(f64(22.8)); got != "73ºF" {
		t.Fatalf("F: %q", got)
	}
	if got := (Opts{Units: UnitC}).Temp(f64(22.8)); got != "23ºC" {
		t.Fatalf("C: %q", got)
	}
	if got := (Opts{Units: UnitF}).Temp(nil); got != "n/a" {
		t.Fatalf("nil: %q", got)
	}
}

func TestHealthGlyphsAreTextual(t *testing.T) {
	o := Opts{Width: 80}
	good := o.HealthGlyph("NWS", snapshot.ProviderOK)
	bad := o.HealthGlyph("NWS", snapshot.ProviderDegraded)
	if !strings.Contains(good, "✔") || !strings.Contains(good, "NWS") {
		t.Fatalf("good glyph: %q", good)
	}
	if !strings.Contains(bad, "✘") && !strings.Contains(bad, "⚠") {
		t.Fatalf("bad glyph must carry a non-color signal: %q", bad)
	}
	// ASCII mode swaps glyphs, never drops the signal (RS-14).
	ascii := (Opts{Width: 80, ASCII: true}).HealthGlyph("NWS", snapshot.ProviderOK)
	if strings.ContainsAny(ascii, "✔⚠✘") || !strings.Contains(ascii, "NWS") {
		t.Fatalf("ascii glyph: %q", ascii)
	}
}

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
	hdr := stripANSI(strings.Split((Opts{Width: 115, Units: UnitF}).LocationTable(nil, 0), "\n")[1])
	for tok, off := range map[string]int{"###.": 11, "NAME": 16, "ZIP": 37, "NOW": 60} {
		if got := runeIdx(hdr, tok); got != off {
			t.Fatalf("%q at %d, want %d\n%q", tok, got, off, hdr)
		}
	}
	if strings.Contains(hdr, "LABEL") {
		t.Fatalf("LABEL must be hidden (UAT 11.2): %q", hdr)
	}
	wide := stripANSI(strings.Split((Opts{Width: 135, Units: UnitF}).LocationTable(nil, 0), "\n")[1])
	if got := runeIdx(wide, "ZIP"); got != 57 {
		t.Fatalf("fill must widen NAME by +20 at 135 cols: ZIP at %d, want 57\n%q", got, wide)
	}
}

func testRow() LocationRow {
	return LocationRow{
		Index: 1, Name: "Oceanside, CA", Tag: "OSIDE", Zip: "92057",
		Conditions: "CLEAR", Now: f64(22.8), Hi: f64(23.9), Lo: f64(17.2), Trend: "up",
		TomorrowConditions: "RAIN", TomorrowHi: f64(25.0), TomorrowLo: f64(18.0),
		HasAlert: true, Selected: true, Playing: true,
	}
}

func TestLocationTableAnatomy(t *testing.T) {
	out := (Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{testRow()}, 0)
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
		{0, "›"}, {3, "▶"}, {9, "⚠"},
		{11, "001."}, {16, "Oceanside, CA"}, {37, "92057"},
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
		return stripANSI((Opts{Width: w, Units: UnitF}).LocationTable([]LocationRow{r}, 0))
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
	out := stripANSI((Opts{Width: 240, Units: UnitF}).LocationTable([]LocationRow{r}, 5))
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
	narrow := stripANSI((Opts{Width: 150, Units: UnitF}).LocationTable([]LocationRow{r}, 5))
	if n := strings.Count(strings.Split(narrow, "\n")[2], "H "); n != 1 {
		t.Fatalf("150 cols fits exactly one extended day, got %d:\n%s", n, narrow)
	}
	if got := (Opts{Width: 240}).TableRowLen(5); got != 240 {
		t.Fatalf("TableRowLen must span the width under fill, got %d", got)
	}
}

func TestKeyCapFallsBackToBrackets(t *testing.T) {
	// With color off (tests run without a tty / NO_COLOR) the chip must
	// degrade to the mock's [key] form — the affordance never disappears
	// (RS-14). Styled output is exercised in live UAT.
	if got := (Opts{}).KeyCap("tab"); got != "[tab]" && !strings.Contains(got, " tab ") {
		t.Fatalf("keycap: %q", got)
	}
	if got := (Opts{ASCII: true}).KeyCap("q"); got != "[q]" {
		t.Fatalf("ascii keycap must be plain brackets: %q", got)
	}
}

func TestPadBetweenIsANSIAware(t *testing.T) {
	line := PadBetween("\x1b[1mleft\x1b[0m", "right", 20)
	if displayWidth(line) != 20 {
		t.Fatalf("padded width = %d, want 20: %q", displayWidth(line), line)
	}
}

func TestAlertBannerSeverityIsText(t *testing.T) {
	a := snapshot.Alert{Event: "Extreme Heat Watch", Severity: "severe", Headline: "until Friday"}
	out := (Opts{Width: 100}).AlertBanner(a, 1, 3)
	if !strings.Contains(out, "[severe]") && !strings.Contains(strings.ToUpper(out), "SEVERE") {
		t.Fatalf("severity must be text (R-12a): %s", out)
	}
	if !strings.Contains(out, "01 / 03") {
		t.Fatalf("alert paging (mock: '01 / 88') missing: %s", out)
	}
}

func TestPanelWidthBound(t *testing.T) {
	out := (Opts{Width: 60}).Panel("Watchpost Weather Radio", "content line")
	if !strings.Contains(out, "Watchpost Weather Radio") {
		t.Fatalf("panel title missing: %s", out)
	}
	for _, line := range strings.Split(stripANSI(out), "\n") {
		if w := displayWidth(line); w > 60 {
			t.Fatalf("panel line %d wide: %q", w, line)
		}
	}
}

func TestConditionVocabularyAndClamping(t *testing.T) {
	// UAT session 3.2: "PARTLY_CLOUDY" (13 cells) overflowed the 12-cell
	// CONDITIONS column and shifted the whole row. The seam maps provider
	// vocabulary to the mock's (P.CLOUDY) and hard-clamps every cell.
	r := testRow()
	r.Conditions = "partly_cloudy"
	r.TomorrowConditions = "MOSTLY_CLOUDY"
	out := stripANSI((Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0))
	lines := strings.Split(out, "\n")
	row := []rune(lines[2])
	if got := string(row[46 : 46+8]); got != "P.CLOUDY" {
		t.Fatalf("condition vocabulary: got %q want P.CLOUDY\n%s", got, out)
	}
	if got := string(row[86 : 86+8]); got != "M.CLOUDY" {
		t.Fatalf("tomorrow condition: got %q\n%s", got, out)
	}
	// A pathological over-wide value must clamp, never shift the row.
	r.Conditions = "SOMETHING_ABSURDLY_LONG_CONDITION"
	out = stripANSI((Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0))
	row = []rune(strings.Split(out, "\n")[2])
	if got := string(row[60 : 60+6]); got != " 73ºF↗" {
		t.Fatalf("NOW must hold col 60 under clamped overflow, got %q\n%s", got, out)
	}
}

func TestColorPassStyling(t *testing.T) {
	// UAT session 3.3/3.4/3.6 with color forced on: HI orange / LO cyan,
	// chips bold-white on light grey, group headers carry their muted
	// backgrounds with the brackets as chip edges.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	if chip := (Opts{}).KeyCap("tab"); !strings.Contains(chip, "48;2;86;86;86") || !strings.Contains(chip, " tab ") {
		t.Fatalf("keycap must be bold white on #565656: %q", chip)
	}
	r := testRow()
	r.Extended = []DayCell{{Date: "08/26", Hi: f64(30.0), Lo: f64(20.0)}}
	out := (Opts{Width: 220, Units: UnitF}).LocationTable([]LocationRow{r}, 1)
	lines := strings.Split(out, "\n")
	for _, code := range []string{"48;2;97;97;97", "48;2;66;94;122", "48;2;66;122;122", "48;2;94;94;122"} {
		if !strings.Contains(lines[0], code) {
			t.Fatalf("group header missing background %s:\n%q", code, lines[0])
		}
	}
	if strings.ContainsAny(stripANSI(lines[0]), "[]") {
		t.Fatalf("brackets are the chip edges — swallowed when styled:\n%q", stripANSI(lines[0]))
	}
	if !strings.Contains(lines[2], "38;5;208") || !strings.Contains(lines[2], "38;5;51") {
		t.Fatalf("row must color HIs orange and LOs cyan:\n%q", lines[2])
	}
	// Styling must never disturb geometry: the row still spans its layout width.
	if w := displayWidth(lines[2]); w > 220 {
		t.Fatalf("styled row overflows: %d", w)
	}
}

func TestSessionFourStyling(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	// 4.8: provider health green when OK, red otherwise.
	o := Opts{Width: 125, Units: UnitF}
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

func TestBlockPaintsFullWidthAndRearms(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	fg, bg := AlertBlockTone("Flash Flood Warning", "moderate")
	o := Opts{Width: 40}
	out := o.Block("hi \x1b[1mchip\x1b[0m tail", fg, bg)
	// UAT 17.2: tile bg hidden (default 49) while the red text tone stays.
	if bg != "49" || !strings.Contains(out, "38;5;196") {
		t.Fatalf("warning block tones (fg red, bg hidden): %q", out)
	}
	// An inner reset must re-arm the block tone, never tear the background.
	if !strings.Contains(out, "\x1b[0;"+fg+";"+bg+"m") {
		t.Fatalf("inner reset must re-arm the block: %q", out)
	}
	if !strings.HasSuffix(out, "\x1b[0m") {
		t.Fatalf("block must close clean: %q", out)
	}
	if w := displayWidth(out); w != 40 {
		t.Fatalf("block must paint the full width, got %d", w)
	}
	afg, abg := AlertBlockTone("Heat Advisory", "minor")
	if afg != "38;5;220" || abg != "49" {
		t.Fatalf("advisory tone (fg yellow, bg hidden): %s %s", afg, abg)
	}
	// Color off: content passes through untinted (text carries the signal).
	rendering.SetColorEnabledForTest(false)
	if plain := o.Block("hello", afg, abg); plain != "hello" {
		t.Fatalf("color-off block must pass through: %q", plain)
	}
	rendering.SetColorEnabledForTest(true)
}

func TestModalBlockKeepsTileBGAfterInnerSpans(t *testing.T) {
	// Session-13 regression: a chip or tint inside a modal line must never
	// drop the tile background for the rest of the line (the reset re-arm
	// must carry BOTH the base fg and the tile bg).
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	o := Opts{Width: 40}
	fg, bg := ModalTone(true)
	line := o.Block("chip "+o.KeyCap("esc")+" end", fg, bg)
	if !strings.Contains(line, "\x1b[0;"+fg+";"+bg+"m") {
		t.Fatalf("inner reset must re-arm fg+tile bg: %q", line)
	}
	if strings.Contains(line, ";49m") || strings.Contains(strings.TrimSuffix(line, "\x1b[0m"), "\x1b[0m"+" ") {
		t.Fatalf("no default-bg fallthrough inside the modal line: %q", line)
	}
}

func TestAlertNameAndTrendTones(t *testing.T) {
	// UAT 14.1/14.2: alerted names muted yellow (advisory) / muted red
	// (warning); trend arrows muted orange up / muted cyan down.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	o := Opts{Width: 125, Units: UnitF}
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
	out := (Opts{Width: 125, Units: UnitF}).LocationTable([]LocationRow{testRow()}, 0)
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
	out := stripANSI((Opts{Width: 150, Units: UnitF}).LocationTable([]LocationRow{r}, 1))
	band := strings.Split(out, "\n")[0]
	if strings.Contains(band, "…") || strings.Contains(band, "EXTENDEDF") {
		t.Fatalf("narrow extended band must use the short title: %q", band)
	}
	if !strings.Contains(band, "E X T E N D E D") {
		t.Fatalf("short title missing: %q", band)
	}
}

func TestThemeTokensAllResolve(t *testing.T) {
	// UAT 16.4: every declared token has a value - a theme swap can never
	// leave a role unpainted.
	for _, tk := range []Token{TextBase, TextBright, TempHi, TempLo, TrendUp, TrendDown,
		FocusName, FocusCell, FocusPointer, NameAdvisory, NameWarning, ProviderOK, ProviderDown, KeyChip, KeyChipMuted, ChipFlashUp, ChipFlashDown, GroupText,
		GroupLocationBG, GroupTodayBG, GroupTomorrowBG, GroupExtendedBG, GroupSectionBG,
		AlertWarnFG, AlertWarnBG, AlertAdvFG, AlertAdvBG, AlertLabel, AlertDanger,
		RadioFG, RadioBG, RadioAccent, StateStopped, StatePlaying, RadioStation, RepeatOn, VizOn,
		ConfirmBG, AlertModalText, AlertModalWarnFG, AlertModalAdvFG, AlertModalWarnBG, AlertModalAdvBG, ModalFG, ModalBGDark, ModalBGLight, WindowBGDark, WindowBGLight,
		GradStart, GradMid, GradEnd} {
		if Tok(tk) == "" {
			t.Fatalf("token %q unresolved", tk)
		}
	}
	if r, g, b := hexRGB(GradStart); r != 0xDD || g != 0x51 || b != 0xD6 {
		t.Fatalf("hexRGB(GradStart) = %d,%d,%d", r, g, b)
	}
}

func TestLoadingShimmerNotNA(t *testing.T) {
	// UAT 18.2b: pending data shimmers through the 4-phase dot sweep; "n/a"
	// is reserved for truly absent data after load.
	r := LocationRow{Index: 1, Name: "Loading City", Zip: "00000", Loading: true}
	for frame, want := range []string{"...", "·..", ".·.", "..·"} {
		out := stripANSI((Opts{Width: 115, Units: UnitF, Frame: frame}).LocationTable([]LocationRow{r}, 0))
		row := strings.Split(out, "\n")[2]
		if !strings.Contains(row, want) {
			t.Fatalf("frame %d must show %q:\n%s", frame, want, row)
		}
		if strings.Contains(row, "n/a") {
			t.Fatalf("loading row must never read n/a:\n%s", row)
		}
	}
	// Post-load nil stays honest.
	loaded := stripANSI((Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{{Index: 1, Name: "X", Zip: "1", Now: f64(20)}}, 0))
	if !strings.Contains(loaded, "n/a") {
		t.Fatalf("post-load missing values must read n/a:\n%s", loaded)
	}
}

func TestModuleInsetFollowsBGVisibility(t *testing.T) {
	// UAT 19.1: visible-bg modules inset 3 cols each side + padded blank
	// top/bottom; hidden-bg modules run flush with no padding lines.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	o := Opts{Width: 40}
	visible := o.Module([]string{"line one"}, "38;5;250", "48;2;48;48;48")
	if got := len(strings.Split(visible, "\n")); got != 3 {
		t.Fatalf("visible-bg module must add padding lines, got %d", got)
	}
	if !strings.Contains(visible, "m   line one") {
		t.Fatalf("visible-bg module must inset content 3 cols: %q", visible)
	}
	hidden := o.Module([]string{"line one"}, "38;5;250", "49")
	if got := len(strings.Split(hidden, "\n")); got != 1 {
		t.Fatalf("hidden-bg module must not pad, got %d lines", got)
	}
	if !strings.Contains(hidden, "mline one") {
		t.Fatalf("hidden-bg module must run flush: %q", hidden)
	}
	if ModuleHeight(7, "49") != 7 || ModuleHeight(7, "48;2;1;1;1") != 9 {
		t.Fatal("ModuleHeight must track visibility")
	}
	if BGVisible("49") || !BGVisible("48;2;48;48;48") {
		t.Fatal("BGVisible classification")
	}
}

func TestAlertBadgeCountAndTone(t *testing.T) {
	// UAT 20: header note icon hidden (column kept); '›  2⚠' badge beside
	// the glyph, toned by the most severe alert (yellow advisory, red
	// warning); go-studs spacing untouched.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	hdr := stripANSI(strings.Split((Opts{Width: 115, Units: UnitF}).LocationTable(nil, 0), "\n")[1])
	if strings.Contains(hdr, "♪") {
		t.Fatalf("header note icon must be hidden (UAT 20.1): %q", hdr)
	}
	warn := testRow()
	warn.HasAlert, warn.WarnAlert, warn.AlertCount, warn.Playing = true, true, 2, false
	adv := testRow()
	adv.Selected, adv.HasAlert, adv.WarnAlert, adv.AlertCount, adv.Playing = false, true, false, 1, false
	out := (Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{warn, adv}, 0)
	lines := strings.Split(out, "\n")
	plain := []rune(stripANSI(lines[2]))
	if got := string(plain[0:11]); got != "›       2⚠ " {
		t.Fatalf("focused 2-alert row must open '›       2⚠ ' (UAT 110 marks block), got %q", got)
	}
	if !strings.Contains(lines[2], "\x1b[38;5;196m2") || !strings.Contains(lines[2], "\x1b[38;5;196m⚠") {
		t.Fatalf("warning badge must be red:\n%q", lines[2])
	}
	if !strings.Contains(lines[3], "\x1b[38;5;220m1") || !strings.Contains(lines[3], "\x1b[38;5;220m⚠") {
		t.Fatalf("advisory badge must be yellow:\n%q", lines[3])
	}
	// Geometry untouched: NAME still lands at col 16 (UAT 110 marks block).
	if got := runeIdx(stripANSI(lines[2]), "Oceanside"); got != 16 {
		t.Fatalf("badge must not disturb column geometry: NAME at %d", got)
	}
}

func TestWrapLinesPreservesIndent(t *testing.T) {
	// UAT 25: the modal component's wrap guarantee — over-wide lines wrap
	// with their indent, blanks pass through, nothing truncates.
	out := WrapLines([]string{
		"",
		"  short",
		"    a very long indented diagnostic line that certainly exceeds the width budget",
	}, 30)
	if len(out) != 6 {
		t.Fatalf("want 6 lines after wrapping, got %d: %q", len(out), out)
	}
	for i, l := range out[2:] {
		if !strings.HasPrefix(l, "    ") {
			t.Fatalf("continuation %d must keep the indent: %q", i, l)
		}
		if displayWidth(l) > 30 {
			t.Fatalf("wrapped line still over-wide: %q", l)
		}
	}
	if strings.Contains(strings.Join(out, " "), "…") {
		t.Fatal("wrap must never truncate")
	}
}

func TestFocusedRowCellsLightBlueAndPointerBoldWhite(t *testing.T) {
	// UAT 50: on the focused row the grey data cells read light blue (name
	// and HI/LO temp tones unchanged) and the pointer is bold white.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	row := strings.Split((Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{testRow()}, 0), "\n")[2]
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
	if r := strings.Split((Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{unfocused}, 0), "\n")[2]; strings.Contains(r, "38;5;117") {
		t.Fatal("unfocused rows must not use the focus cell tone")
	}
}

func TestThemeRegistrySwitchesLive(t *testing.T) {
	// UAT 53: themes register as overrides over the default; SetTheme flips
	// the live table so Tok() resolves the new value; unknown names refuse.
	defer SetTheme(DefaultThemeName)
	if got := ThemeNames()[0]; got != DefaultThemeName {
		t.Fatalf("default must list first, got %q", got)
	}
	RegisterTheme("Test Ember", map[Token]string{TempHi: "196"})
	if !SetTheme("Test Ember") || Tok(TempHi) != "196" {
		t.Fatalf("SetTheme must resolve live: %q", Tok(TempHi))
	}
	if Tok(TempLo) != "51" {
		t.Fatal("unlisted tokens inherit the default")
	}
	if SetTheme("nope") || ThemeName() != "Test Ember" {
		t.Fatal("unknown theme must be refused and leave the active theme")
	}
	for _, n := range []string{"High Contrast", "Monochrome", "Solarized Night", "Synthwave '84"} {
		if !SetTheme(n) {
			t.Fatalf("built-in %q must register", n)
		}
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
		return strings.Split(stripANSI((Opts{Width: w, Units: u}).LocationTable([]LocationRow{r}, 0)), "\n")
	}
	full := at(131, UnitF)
	for tok, off := range map[string]int{"NAME": 16, "WX STN": 37, "DIST": 45, "ZIP": 53, "CONDITIONS": 62} {
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
	if out := stripANSI((Opts{Width: 131, Units: UnitF}).LocationTable([]LocationRow{blank}, 0)); strings.Contains(out, " mi") {
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
	// Very narrow: ZIP leaves last, NAME keeps its 10-cell floor (at 50
	// cols the UAT 110 marks block leaves NAME exactly its floor, so the
	// full name survives from 55).
	narrow := at(50, UnitF)
	if strings.Contains(narrow[1], "ZIP") || strings.Contains(narrow[2], "92057") {
		t.Fatalf("50 cols must drop ZIP before breaking NAME:\n%s", strings.Join(narrow, "\n"))
	}
	if !strings.Contains(narrow[2], "Oceansi") || !strings.Contains(narrow[2], " 73ºF") {
		t.Fatalf("NAME and NOW survive every width:\n%s", strings.Join(narrow, "\n"))
	}
	if n55 := at(55, UnitF); !strings.Contains(n55[2], "Oceanside") {
		t.Fatalf("55 cols carries the full name:\n%s", strings.Join(n55, "\n"))
	}
	for _, l := range narrow {
		if w := displayWidth(l); w > 50 {
			t.Fatalf("line exceeds 50: %q", l)
		}
	}
}

func TestUntitledPanelHasUnbrokenTopRule(t *testing.T) {
	// UAT 68: the About window has no title; the top rule must not carry
	// the empty title's spaces.
	top := strings.Split((Opts{Width: 20}).PanelColored("", "x", ""), "\n")[0]
	if strings.Contains(top, " ") || displayWidth(top) != 20 {
		t.Fatalf("untitled top rule: %q", top)
	}
}

func TestFireMarkIsACountedOrangeDiamond(t *testing.T) {
	// UAT 110 mock `›  ▶ 3◆ 2⚠ 001.`: fire is a counted orange ◆ in its own
	// slot pair (bold when a hotspot burns hard), * under --ascii, never
	// displacing the play or alert marks; NAME's floor gives up the cells.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	r := testRow()
	r.Fire, r.FireHot, r.AlertCount = 3, true, 2
	out := (Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0)
	plain := []rune(stripANSI(strings.Split(out, "\n")[2]))
	if got := string(plain[0:15]); got != "›  ▶ 3◆ 2⚠ 001." {
		t.Fatalf("marks block: %q", got)
	}
	if bold := Tint("◆", "1;"+Tok(FireMark)); !strings.Contains(out, bold) {
		t.Fatalf("a hot row wears a bold ◆:\n%q", out)
	}
	r.FireHot = false
	if out := (Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0); strings.Contains(out, Tint("◆", "1;"+Tok(FireMark))) {
		t.Fatal("a cool row must not read bold")
	}
	ascii := stripANSI((Opts{Width: 115, Units: UnitF, ASCII: true}).LocationTable([]LocationRow{r}, 0))
	if !strings.Contains(ascii, "3*") || strings.Contains(ascii, "◆") {
		t.Fatalf("--ascii swaps ◆ for *:\n%s", ascii)
	}
	r.Fire = 0
	if strings.Contains(stripANSI((Opts{Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0)), "◆") {
		t.Fatal("no fire, no mark")
	}
}

func TestEveryThemeOwnsItsTitleGradient(t *testing.T) {
	// UAT 107 (HUM LEAD): the wordmark gradient follows the theme — every
	// built-in sets all three stops itself, and they differ from the default
	// (a new theme that forgets them fails here, not at UAT).
	def := builtinOverrides()
	for name, over := range def {
		for _, tok := range []Token{GradStart, GradMid, GradEnd} {
			v, ok := over[tok]
			if !ok || len(v) != 7 || v[0] != '#' {
				t.Fatalf("theme %q must set %s as #RRGGBB, got %q", name, tok, v)
			}
		}
		if def := defaultTheme(); over[GradStart] == def[GradStart] && over[GradMid] == def[GradMid] && over[GradEnd] == def[GradEnd] {
			t.Fatalf("theme %q reuses the default gradient", name)
		}
	}
}

// validRawSGR reports whether a composite token value is a legal SGR
// parameter list for the raw path (sgrRaw emits it verbatim): attributes,
// basic colours, and fully-qualified 38;5;n / 48;5;n / 38;2;r;g;b /
// 48;2;r;g;b — never a bare 256-palette index, which the terminal ignores.
func validRawSGR(v string) bool {
	e := strings.Split(v, ";")
	for i := 0; i < len(e); i++ {
		if (e[i] == "38" || e[i] == "48") && i+1 < len(e) {
			switch e[i+1] {
			case "5":
				i += 2
				continue
			case "2":
				i += 4
				continue
			}
		}
		n, err := strconv.Atoi(e[i])
		if err != nil || !legalSGRAttr(n) {
			return false
		}
	}
	return true
}

// legalSGRAttr: attributes, basic and bright colours, defaults.
func legalSGRAttr(n int) bool {
	return n <= 9 || (n >= 21 && n <= 29) || (n >= 30 && n <= 37) || n == 39 || (n >= 40 && n <= 47) || n == 49 || (n >= 90 && n <= 97) || (n >= 100 && n <= 107)
}

func TestCompositeThemeTokensAreLegalRawSGR(t *testing.T) {
	// UAT 108 (High Contrast chips read white on a light chip): a composite
	// token rides the raw path, so a bare palette index inside it ("1;16;…")
	// is silently dropped by the terminal. Every composite value in every
	// theme must be fully qualified.
	check := func(theme string, table map[Token]string) {
		for tok, v := range table {
			raw := strings.HasPrefix(v, "38;") || strings.HasPrefix(v, "48;") || strings.Contains(v, ";38;") || strings.Contains(v, ";48;") // the raw path: composite colour lists (Tint-path tokens like "1;220" are qualified by SGR())
			if raw && !validRawSGR(v) {
				t.Errorf("theme %q token %s = %q is not a legal raw SGR list (qualify palette codes as 38;5;n / 48;5;n)", theme, tok, v)
			}
		}
	}
	check(DefaultThemeName, defaultTheme())
	for name, over := range builtinOverrides() {
		check(name, over)
	}
}

// Quality pass Q0: the byte column of the [S] REQUESTS rows never exceeds
// six cells, whatever the count.
func TestHumanBytesFitsSixCells(t *testing.T) {
	for n, want := range map[int64]string{0: "0B", 1023: "1023B", 1024: "1.0K", 12_900_000: "12.3M", 1023 << 20: "1023M", 1 << 30: "1.0G"} {
		if got := HumanBytes(n); got != want || len(got) > 6 {
			t.Fatalf("HumanBytes(%d) = %q, want %q (≤ 6 cells)", n, got, want)
		}
	}
}

// The mock's two header lines (test-only: the goldens pin the render against them).
const (
	mockGroupHeader = "           [   L O C A T I O N   /   S T A T I O N    ]     [           T O D A Y             ]     [    T O M O R R O W     ]"
	mockColHeader   = "  ♪        ###. NAME                 LABEL    ZIP      CONDITIONS    NOW       HI     LOW      CONDITIONS    HI      LOW"
)
