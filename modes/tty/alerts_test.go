package tty

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestAlertPagingKeys(t *testing.T) {
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = append(s2.Locations[0].Alerts,
		snapshot.Alert{Event: "Flash Flood Warning", Severity: "severe", Headline: "second"})
	m, _ = m.Update(SnapshotMsg{Snap: s2})
	if v := m.View().Content; !strings.Contains(v, "01 / 02 Alerts") {
		t.Fatalf("pager: %s", v)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v := m.View().Content; !strings.Contains(v, "02 / 02 Alerts") || !strings.Contains(v, "FLASH FLOOD WARNING") {
		t.Fatalf("right must page to alert 2:\n%s", v)
	}
}

func TestAlertAreaHeightIsFixed(t *testing.T) {
	// UAT 5.2: the alert area stays reserved-but-blank when the focused
	// location has no alert, so the layout below it never jumps.
	m := dash(t)
	withAlert := strings.Count(m.View().Content, "\n")
	s2 := snap()
	s2.Locations[0].Alerts = nil
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	withoutAlert := strings.Count(m2.View().Content, "\n")
	if withAlert != withoutAlert {
		t.Fatalf("alert area must reserve its space: %d lines with alert, %d without", withAlert, withoutAlert)
	}
	if v := m2.View().Content; strings.Contains(v, "EXTREME HEAT WATCH") {
		t.Fatal("blank reservation must not render alert text")
	}
}

func TestAlertTitleNamesTheLocation(t *testing.T) {
	// UAT 5.3: the module's inner text matches the focused location.
	v := dash(t).View().Content
	if !strings.Contains(v, "EXTREME HEAT WATCH · Oceanside, CA") {
		t.Fatalf("alert title must carry the focused location:\n%s", v)
	}
}

func TestAlertBodyWrapsInFixedThreeLineArea(t *testing.T) {
	// UAT 15.2/15.2a: long alert bodies wrap (never truncate with …) inside
	// a constant 3-line body area, so the module height never bounces.
	m := dash(t)
	long := snap()
	long.Locations[0].Alerts = []snapshot.Alert{{Event: "Extreme Heat Watch", Severity: "severe",
		Headline: "dangerously hot conditions with temperatures up to 112 expected across the inland valleys through Friday evening"}}
	m2, _ := m.Update(SnapshotMsg{Snap: long})
	m2, _ = m2.Update(tea.WindowSizeMsg{Width: 100, Height: 60}) // narrow: forces the wrap
	v := m2.View().Content
	if strings.Contains(v, "…") {
		t.Fatalf("alert body must wrap, not truncate:\n%s", v)
	}
	if !strings.Contains(v, "inland valleys") {
		t.Fatalf("wrapped continuation must render:\n%s", v)
	}
	short, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 60})
	if a, b := strings.Count(short.View().Content, "\n"), strings.Count(v, "\n"); a != b {
		t.Fatalf("module height must not bounce between short (%d) and wrapped (%d) bodies", a, b)
	}
}

func TestAlertsSortMostSevereFirst(t *testing.T) {
	// UAT 16.2: index 0 is always the worst active alert - the module's
	// first page and the name tint agree (Los Angeles case).
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{
		{Event: "Heat Advisory", Severity: "minor", Headline: "advisory first in feed"},
		{Event: "Flash Flood Warning", Severity: "severe", Headline: "warning buried second"},
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	if got := d.snap.Locations[0].Alerts[0].Event; got != "Flash Flood Warning" {
		t.Fatalf("most severe must sort first, got %q", got)
	}
	if v := m2.View().Content; !strings.Contains(v, "FLASH FLOOD WARNING") {
		t.Fatalf("module must open on the severe alert:\n%s", v)
	}
}

func TestPagingChipsMuteWhenInert(t *testing.T) {
	// UAT 21.1: a chip whose press would do nothing reads muted; state
	// flows from the model (ELM) through render.KeyCapIf.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	muted, full := "48;2;43;43;43", "48;2;86;86;86"

	one := dash(t) // single alert: both paging chips inert
	v := one.View().Content
	if got := strings.Count(v, muted); got != 2 {
		t.Fatalf("single-alert view must mute both paging chips, got %d muted", got)
	}
	two := snap()
	two.Locations[0].Alerts = append(two.Locations[0].Alerts,
		snapshot.Alert{Event: "Flash Flood Warning", Severity: "severe", Headline: "second"})
	m, _ := one.Update(SnapshotMsg{Snap: two})
	if v := m.View().Content; strings.Count(v, muted) != 1 || !strings.Contains(v, full) {
		// at index 0 of 2: ← inert, → live
		t.Fatalf("2-alert idx0 must mute only ←:\n%d muted", strings.Count(v, muted))
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v := m.View().Content; strings.Count(v, muted) != 1 {
		t.Fatalf("2-alert idx1 must mute only →: %d muted", strings.Count(v, muted))
	}
}

func TestAKeyOpensAlertDetailsModal(t *testing.T) {
	// UAT 22: [A] floats the full alert record — event + severity/urgency/
	// certainty, timing with duration, area, description, instruction —
	// on a severity-tinted tile.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	onset := time.Date(2026, 8, 24, 11, 0, 0, 0, time.UTC)
	ends := onset.Add(30 * time.Hour)
	s2.Locations[0].Alerts = []snapshot.Alert{{
		Event: "Extreme Heat Watch", Severity: "severe", Urgency: "expected", Certainty: "likely",
		Headline: "until Friday", Onset: &onset, Ends: &ends, Effective: onset, Expires: ends,
		AreaDesc:    "San Diego County Coastal Areas",
		Description: "Dangerously hot conditions with temperatures up to 112 possible.",
		Instruction: "Drink plenty of fluids and stay out of the sun.",
	}}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	v := m2.View().Content
	for _, want := range []string{
		"ALERT 1 / 1 · Oceanside, CA",
		"EXTREME HEAT WATCH", "[severe · expected · likely]",
		"Starts", "Ends", "(~30h0m0s)",
		"San Diego County", "Dangerously hot", "Instructions: Drink plenty",
	} {
		if !strings.Contains(v, want) {
			t.Fatalf("alert modal missing %q:\n%s", want, v)
		}
	}
	if !strings.Contains(v, "48;2;40;0;0") {
		t.Fatal("warning-grade modal must carry the muted red tile")
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v := m3.View().Content; strings.Contains(v, "ALERT 1 / 1") {
		t.Fatal("esc must close the alert modal")
	}
}

func TestAlertModalPagesWithTonedTiles(t *testing.T) {
	// UAT 23: left/right page alerts inside the modal (no esc round-trip);
	// the tile re-tints to the FOCUSED alert's class per page.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{
		{Event: "Heat Advisory", Severity: "minor", Headline: "advisory"},
		{Event: "Flash Flood Warning", Severity: "severe", Headline: "warning"},
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2}) // sorts: warning first
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	v := m2.View().Content
	if !strings.Contains(v, "ALERT 1 / 2") || !strings.Contains(v, "48;2;40;0;0") {
		t.Fatalf("page 1 must be the warning on the red tile:\n%s", v)
	}
	m2, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	v = m2.View().Content
	if !strings.Contains(v, "ALERT 2 / 2") || !strings.Contains(v, "48;2;32;28;0") || !strings.Contains(v, "HEAT ADVISORY") {
		t.Fatalf("page 2 must be the advisory on the yellow tile:\n%s", v)
	}
	if !strings.Contains(v, "48;2;43;43;43") {
		t.Fatal("inert → chip must read muted on the last page")
	}
	m2, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v := m2.View().Content; !strings.Contains(v, "ALERT 1 / 2") {
		t.Fatal("left must page back without closing")
	}
}

func TestAlertBulletFormattingRules(t *testing.T) {
	// location-detail-mock.txt bullet rules: prose indents 2 and bullets 4
	// from the text edge (flush with the section labels, UAT 65);
	// single-line bullets stack tight; multi-line bullets get one blank
	// line above and below.
	desc := "WHAT...Dangerous heat.\n\n* FIRST SHORT BULLET\n* SECOND SHORT BULLET\n* " +
		"A VERY LONG BULLET THAT WILL CERTAINLY WRAP ACROSS MULTIPLE LINES WHEN CONSTRAINED TO A NARROW WIDTH BUDGET FOR THE MODAL\n* TAIL SHORT BULLET"
	out := formatAlertBody(desc, 60)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "  WHAT...Dangerous heat.") {
		t.Fatalf("prose paragraph missing:\n%s", joined)
	}
	i1 := strings.Index(joined, "- FIRST")
	i2 := strings.Index(joined, "- SECOND")
	if strings.Count(joined[i1:i2], "\n") != 1 {
		t.Fatalf("adjacent single-line bullets must stack tight:\n%s", joined)
	}
	iLong := strings.Index(joined, "- A VERY LONG")
	if !strings.Contains(joined[:iLong], "- SECOND SHORT BULLET\n\n") {
		t.Fatalf("multi-line bullet needs a blank above:\n%s", joined)
	}
	if !strings.Contains(joined, "MODAL\n\n    - TAIL") {
		t.Fatalf("multi-line bullet needs a blank below:\n%s", joined)
	}
	for _, l := range out {
		if strings.HasPrefix(l, "    - ") || strings.HasPrefix(l, "      ") || strings.HasPrefix(l, "  ") || l == "" {
			continue
		}
		t.Fatalf("unexpected line shape: %q", l)
	}
}

func TestModalAlertTonesAndBoldTitles(t *testing.T) {
	// UAT 28.3-5: advisory #ACAE7D / warning #BE5454 text in the modals,
	// titles bold.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{
		{Event: "Flash Flood Warning", Severity: "severe", Description: "Warning prose."},
		{Event: "Heat Advisory", Severity: "minor", Description: "Advisory prose."},
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	v := m2.View().Content
	// The compositor may reorder params (bold before or after the color).
	warnTitle := regexp.MustCompile(`\x1b\[(1;)?38;2;190;84;84(;1)?m⚠ FLASH FLOOD WARNING`)
	advTitle := regexp.MustCompile(`\x1b\[(1;)?38;2;172;174;125(;1)?m⚠ HEAT ADVISORY`)
	if !warnTitle.MatchString(v) {
		t.Fatalf("warning title must be bold #BE5454:\n%s", v)
	}
	if !advTitle.MatchString(v) {
		t.Fatalf("advisory title must be bold #ACAE7D:\n%s", v)
	}
	if !strings.Contains(v, "\x1b[38;2;172;174;125m  Advisory prose.") {
		t.Fatalf("advisory body must carry the advisory tone:\n%s", v)
	}
}

func TestAlertModalBodyTextIsWhite(t *testing.T) {
	// UAT 55: in the [A] modal the body text is white; the title keeps its tone.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{{Event: "Heat Advisory", Severity: "minor", Headline: "h",
		AreaDesc: "Coastal Areas", Description: "Hot afternoon.", Instruction: "Hydrate."}}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	v := m2.View().Content
	for _, want := range []string{"\x1b[97m  Area: Coastal Areas", "\x1b[97m  Hot afternoon.", "\x1b[97m  Instructions: Hydrate."} {
		if !strings.Contains(v, want) {
			t.Fatalf("body text must be white: missing %q\n%s", want, v)
		}
	}
	if !regexp.MustCompile(`\x1b\[(1;)?38;2;172;174;125(;1)?m`).MatchString(v) {
		t.Fatal("title keeps its advisory tone")
	}
}
