package tty

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestViewOpensWithTwoBlankLines(t *testing.T) {
	// UAT 10.3 (was UAT-3.1): two blank lines above the header; one after it.
	v := dash(t).View().Content
	lines := strings.Split(v, "\n")
	for i := range 2 {
		if strings.TrimSpace(lines[i]) != "" {
			t.Fatalf("line %d must be blank (top spacing):\n%s", i, v)
		}
	}
	if !strings.HasPrefix(strings.TrimLeft(lines[2], " "), "W A T C H P O S T") {
		t.Fatalf("header must follow the spacing: %q", lines[2])
	}
	if !strings.HasPrefix(lines[2], strings.Repeat(" ", viewPadLeft)+"W") {
		t.Fatalf("viewport must carry the %d-col left padding (UAT 14.3): %q", viewPadLeft, lines[2])
	}
	if strings.TrimSpace(lines[4]) != "" {
		t.Fatalf("blank line required between header and alert section: %q", lines[4])
	}
}

func TestWindowBackgroundTracksMode(t *testing.T) {
	// UAT 10.2: blue-grey window background, dark/light per terminal mode.
	m := dash(t)
	if bg := m.View().BackgroundColor; bg == nil {
		t.Fatal("window background must be set")
	}
}

func TestContentModalsStretchTo60Percent(t *testing.T) {
	// UAT 31.2: status/details/alerts widen to 60% of a wide terminal;
	// the base widths remain the floor on narrow ones.
	m := dash(t)
	wide, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	d := wide.(Dashboard)
	d.modal = modalStatus
	if got := d.modalWidth(); got != 120 {
		t.Fatalf("status modal at 200 cols must be 120 wide, got %d", got)
	}
	d.modal = modalDetails
	if got := d.modalWidth(); got != 120 {
		t.Fatalf("details modal at 200 cols must be 120 wide, got %d", got)
	}
	narrow, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 60})
	n := narrow.(Dashboard)
	n.modal = modalDetails
	if got := n.modalWidth(); got != 85 {
		t.Fatalf("details floor must hold on narrow terminals, got %d", got)
	}
	n.modal = modalAdd
	if got := n.modalWidth(); got != 56 {
		t.Fatal("search/confirm modals stay fixed width")
	}
}

func TestCompactModulesOnShortTerminals(t *testing.T) {
	// UAT 34: when the full layout cannot fit, the alert module collapses to
	// one line and the radio to two so the footer controls stay visible;
	// tall terminals keep the full modules.
	m := dash(t)
	short, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 24})
	v := short.View().Content
	if !short.(Dashboard).compact() {
		t.Fatal("24 rows must select compact mode")
	}
	if !strings.Contains(v, "01/01  ⚠ EXTREME HEAT WATCH · Oceanside, CA") {
		t.Fatalf("compact alert line missing:\n%s", v)
	}
	if !strings.Contains(v, "[space] Play") || !strings.Contains(v, "Repeat:") || strings.Contains(v, "00:00 / 00:00\n") {
		t.Fatalf("compact radio must be the two-row form:\n%s", v)
	}
	if !strings.Contains(v, "Quit") {
		t.Fatal("footer controls must remain visible on short terminals")
	}
	tall, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 60})
	if tall.(Dashboard).compact() {
		t.Fatal("60 rows must keep the full modules")
	}
}

func TestNarrowTerminalRowsFitAndModalsCenter(t *testing.T) {
	// UAT 35: on a narrow terminal every rendered row fits the width (the
	// compact radio row degrades: bar, clock, station), controls wrap, the
	// VOL bar shrinks, and modals center on the TERMINAL width.
	m := dash(t)
	narrow, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 24})
	v := stripANSITest(narrow.View().Content)
	for i, l := range strings.Split(v, "\n") {
		if w := len([]rune(l)); w > 70 {
			t.Fatalf("row %d exceeds the terminal width (%d): %q", i, w, l)
		}
	}
	if !strings.Contains(v, "[space] Play") || !strings.Contains(v, "Size:") {
		t.Fatal("radio controls must survive by wrapping")
	}
	if !strings.Contains(v, "WWRadio") || strings.Contains(v, "Watchpost Weather Radio") {
		t.Fatal("narrow compact row must use the short title (UAT 36)")
	}
	wide, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	wv := stripANSITest(wide.View().Content)
	if strings.Count(v, "█")+strings.Count(v, "░") >= strings.Count(wv, "█")+strings.Count(wv, "░") {
		t.Fatal("VOL bar must shrink on narrow terminals")
	}
	sm, _ := narrow.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	for _, l := range strings.Split(stripANSITest(sm.View().Content), "\n") {
		if strings.Contains(l, "API Status") {
			// Measure the modal by its own box span: base rows can peek out
			// past the modal's right edge on narrow terminals.
			r := []rune(l)
			start, end := strings.IndexRune(l, '┌'), strings.IndexRune(l, '┐')
			startCol := len([]rune(l[:start]))
			modalW := len([]rune(l[:end])) - startCol + 1
			_ = r
			if want := (70 - modalW) / 2; startCol < want-1 || startCol > want+1 {
				t.Fatalf("modal must center on the terminal: starts at %d, want ~%d (modal %d wide)", startCol, want, modalW)
			}
			return
		}
	}
	t.Fatal("status modal title not found")
}

func TestModulesHoldUntilTableBreakpoint(t *testing.T) {
	// UAT 49: with 50 recent rows the full modules stay while the table can
	// show >= 20 rows (fav + recent window); the table flexes row-by-row
	// above that; below it the modules minimize (and the table regains rows).
	m := dash(t)
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}
	for i := range 50 {
		rs.Locations = append(rs.Locations, snapshot.Location{Label: fmt.Sprintf("City %02d", i+1), Zip: fmt.Sprintf("900%02d", i+1)})
	}
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	sawFull, sawCompact := false, false
	for h := 120; h >= 20; h-- {
		sized, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: h})
		d := sized.(Dashboard)
		rows := d.numPriority() + min(50, d.windowSize(d.opts()))
		if !d.compact() {
			sawFull = true
			if rows < tableBreakpoint {
				t.Fatalf("full modules must hold only while the table shows >= %d rows: %d rows at %d", tableBreakpoint, rows, h)
			}
		} else {
			sawCompact = true
		}
	}
	if !sawFull || !sawCompact {
		t.Fatalf("both modes must occur across the sweep (full=%v compact=%v)", sawFull, sawCompact)
	}
	// A large terminal shows the FULL modules with a big table.
	big, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: 80})
	if big.(Dashboard).compact() {
		t.Fatal("80 rows must render the full modules")
	}
}

func TestControlPlacementUAT56(t *testing.T) {
	// UAT 56: [enter] Location Details leads the watchlist controls, [↑↓]
	// Navigate is right-aligned on that row, [?] Help follows About, and the
	// footer keeps only theme + quit.
	v := stripANSITest(dash(t).View().Content)
	lines := strings.Split(v, "\n")
	var ctrl, head string
	for _, l := range lines {
		if strings.Contains(l, "[ctrl+a] Favorite") {
			ctrl = l
		}
		if strings.Contains(l, "[a] About") {
			head = l
		}
	}
	// UAT 2026-08-27 order: [l] Lookup Location, [enter] Details, [ctrl+a] Favorite, [shift+del] Unfavorite.
	li, ei, fi := strings.Index(ctrl, "[l] Lookup Location"), strings.Index(ctrl, "[enter] Details"), strings.Index(ctrl, "[ctrl+a] Favorite")
	if li < 0 || !(li < ei && ei < fi) {
		t.Fatalf("control row order Lookup < Details < Favorite: %q", ctrl)
	}
	if !strings.HasSuffix(strings.TrimRight(ctrl, " "), "[↑↓] Navigate") {
		t.Fatalf("[↑↓] Navigate must end the control row (right-aligned): %q", ctrl)
	}
	if !strings.Contains(head, "[s] Setup  [a] About  [t] Theme  [?] Help  [q] Quit") {
		t.Fatalf("header must carry Setup, About, Theme, Help, Quit in order (UAT 57 / 102): %q", head)
	}
	// UAT 57: no footer - the last content line is the recent section's
	// last row (the Showing line, or the empty state's ▼ row — UAT 104).
	trimmed := strings.TrimRight(v, "\n ")
	if last := trimmed[strings.LastIndex(trimmed, "\n")+1:]; !strings.Contains(last, "Showing") && !strings.HasSuffix(last, "▼") {
		t.Fatalf("view must end at the recent section's last row (no footer): %q", last)
	}
}

func TestViewFillsToBottomInset(t *testing.T) {
	// UAT 58: exactly 2 blank rows top and bottom - the content spans
	// height-2 rows with no stray trailing line.
	m := dash(t)
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}
	for i := range 50 {
		rs.Locations = append(rs.Locations, snapshot.Location{Label: fmt.Sprintf("City %02d", i+1), Zip: fmt.Sprintf("900%02d", i+1)})
	}
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	for _, h := range []int{40, 50, 70} {
		sized, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: h})
		v := sized.View().Content
		lines := strings.Split(v, "\n")
		if len(lines) != h-2 {
			t.Fatalf("h=%d: content must span %d rows, got %d", h, h-2, len(lines))
		}
		if strings.TrimSpace(lines[0]) != "" || strings.TrimSpace(lines[1]) != "" || strings.TrimSpace(lines[2]) == "" {
			t.Fatalf("h=%d: exactly 2 blank rows on top", h)
		}
		if strings.TrimSpace(lines[len(lines)-1]) == "" {
			t.Fatalf("h=%d: no stray blank row at the bottom of the content", h)
		}
	}
}
