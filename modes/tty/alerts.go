package tty

// alerts.go — the alert modal and the alert area under the header. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	zones "github.com/branden-thompson/watchpost/platform/tz"
)

// alertBlocks renders each alert in full with the mock's bullet rules,
// separated by dividers.
func alertBlocks(loc *snapshot.Location, w int) []string {
	if len(loc.Alerts) == 0 {
		return nil
	}
	// UAT 33.1/65/66: the divider and the alert text run from the text edge
	// (flush with the section labels) to 3 cells before the scroll rail:
	// w is the modal width less 11; the rail sits at w+5, so the right
	// edge is w+2.
	textW := w + 2
	divider := strings.Repeat("─", textW)
	lines := []string{}
	for _, a := range loc.Alerts {
		tone := modalAlertTone(a)
		lines = append(lines, "", divider, "", render.TintRaw("⚠ "+strings.ToUpper(render.Plain(a.Event)), "1;"+tone)) // bold title (UAT 28.5)
		for _, l := range formatAlertBody(render.Plain(a.Description), textW) {
			lines = append(lines, render.TintRaw(l, tone))
		}
	}
	return lines
}

// modalAlertTone: advisory #ACAE7D / warning #BE5454 text in modals
// (UAT 28.3/28.4) - theme tokens on the raw-SGR path.
func modalAlertTone(a snapshot.Alert) string {
	if render.AlertIsWarning(a.Event, a.Severity) {
		return render.Tok(render.AlertModalWarnFG)
	}
	return render.Tok(render.AlertModalAdvFG)
}

// formatAlertBody applies the mock's bullet rules to NWS alert prose:
// prose indents 2 and "*"-bullets 4 cols from the text edge (the flush
// section-label column, UAT 65); single-line bullets stack tight; a
// multi-line bullet gets one blank line above and below. textW is the
// full line budget — every line, indent included, ends by it (UAT 66).
func formatAlertBody(desc string, textW int) []string {
	out := []string{}
	prevMulti := false
	for _, item := range splitAlertItems(desc) {
		if !item.bullet {
			for _, l := range render.WrapText(item.text, textW-2) {
				out = append(out, "  "+l)
			}
			prevMulti = false
			continue
		}
		wrapped := render.WrapText(item.text, textW-6)
		multi := len(wrapped) > 1
		if (multi || prevMulti) && len(out) > 0 {
			out = append(out, "")
		}
		for j, l := range wrapped {
			prefix := "    - "
			if j > 0 {
				prefix = "      "
			}
			out = append(out, prefix+l)
		}
		prevMulti = multi
	}
	return out
}

// alertItem is one prose paragraph or bullet from an NWS description.
type alertItem struct {
	text   string
	bullet bool
}

// splitAlertItems parses NWS description text: paragraphs separated by
// blank lines; "* "-prefixed items are bullets.
func splitAlertItems(desc string) []alertItem {
	items := []alertItem{}
	for _, para := range strings.Split(desc, "\n\n") {
		para = strings.TrimSpace(strings.ReplaceAll(para, "\n", " "))
		if para == "" {
			continue
		}
		for i, piece := range strings.Split(para, "* ") {
			piece = strings.TrimSpace(piece)
			if piece == "" {
				continue
			}
			items = append(items, alertItem{text: piece, bullet: i > 0 || strings.HasPrefix(para, "* ")})
		}
	}
	return items
}

// prettyCond renders a condition code as title-cased prose ("Partly Cloudy").
func prettyCond(c string) string {
	words := strings.Fields(strings.ToLower(strings.ReplaceAll(c, "_", " ")))
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	if len(words) == 0 {
		return "--"
	}
	return strings.Join(words, " ")
}

// truncateTo hard-limits a plain string to n cells.
func truncateTo(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// alertDetailsModal floats ONE alert at a time (UAT 23.2): the tile tint
// follows the FOCUSED alert's class (yellow advisory / red warning) as you
// page, keeping attention on each individual statement.
func (d Dashboard) alertDetailsModal(o render.Opts) string {
	sel := d.selectedLocation()
	fg, _ := render.ModalTone(d.darkBG)
	if sel == nil || len(sel.Alerts) == 0 {
		return d.floatModalToned(o, d.modalWidth(), "ALERTS", d.alertDetailLines(), fg, render.Tok(render.AlertModalAdvBG))
	}
	a := sel.Alerts[d.alertIdx%len(sel.Alerts)]
	bg := render.Tok(render.AlertModalAdvBG)
	if render.AlertIsWarning(a.Event, a.Severity) {
		bg = render.Tok(render.AlertModalWarnBG)
	}
	title := fmt.Sprintf("ALERT %d / %d · %s", d.alertIdx+1, len(sel.Alerts), sel.Label)
	return d.floatModalToned(o, d.modalWidth(), title, d.alertDetailLines(), fg, bg)
}

// alertDetailLines renders the FOCUSED alert's full record plus the paging
// controls (chips mute per direction - UAT 23.1).
func (d Dashboard) alertDetailLines() []string {
	sel := d.selectedLocation()
	o := d.opts()
	if sel == nil || len(sel.Alerts) == 0 {
		return []string{"", "  No active alerts for this location.", "", "  " + o.KeyCap("esc") + " Close"}
	}
	n := len(sel.Alerts)
	idx := d.alertIdx % n
	wrapW := min(o.Width, d.modalWidth()) - 9 // breathing room beside the scroll rail (UAT 23.3)
	lines := []string{""}
	lines = append(lines, alertRecordLines(o, idx, sel.Alerts[idx], wrapW, alertClock(sel))...)
	controls := o.KeyCapIf("←", idx > 0) + " Previous  " + o.KeyCapIf("→", idx < n-1) + " Next   " +
		o.KeyCap("esc") + " Close   " + o.KeyCap("↑↓") + " Scroll"
	return append(lines, "  "+controls)
}

// dataAsOf is the header's "Last Updated" time (red-team 0.9.0 F7): the
// newest successful provider fetch — not the publish time, which keeps
// ticking while every row is stale last-good data offline.
func dataAsOf(sn *snapshot.Snapshot) time.Time {
	if sn == nil {
		return time.Time{}
	}
	at := time.Time{}
	for _, p := range sn.Providers {
		if p.FetchedAt.After(at) {
			at = p.FetchedAt
		}
	}
	if at.IsZero() {
		return sn.GeneratedAt
	}
	return at
}

// alertClock is the zone an alert's times are shown in: the location's
// own (F17 — a Miami alert is not on Pacific time), else the machine's.
func alertClock(loc *snapshot.Location) *time.Location {
	if loc != nil && loc.TZ != "" {
		if z, err := zones.Location(loc.TZ); err == nil {
			return z
		}
	}
	return time.Local
}

// alertRecordLines formats one alert's full record for the modal.
func alertRecordLines(o render.Opts, i int, a snapshot.Alert, wrapW int, in *time.Location) []string {
	tone := modalAlertTone(a)                      // UAT 28.3/28.4 modal text tones
	_ = i                                          // paging lives in the modal title now (UAT 23.2)
	head := strings.ToUpper(render.Plain(a.Event)) // provider text never addresses the terminal (S-F6)
	meta := fmt.Sprintf("[%s · %s · %s]", a.Severity, a.Urgency, a.Certainty)
	out := []string{"  " + render.TintRaw(head, "1;"+tone) + "  " + meta} // bold title (UAT 28.5)
	start, end := a.Effective, a.Expires
	if a.Onset != nil {
		start = *a.Onset
	}
	if a.Ends != nil {
		end = *a.Ends
	}
	timing := "  Starts " + start.In(in).Format("Mon 01/02 3:04 PM") // the location's clock (F17)
	if end.After(start) {
		timing += "   Ends " + end.In(in).Format("Mon 01/02 3:04 PM") +
			fmt.Sprintf("   (~%s)", end.Sub(start).Round(time.Hour))
	}
	out = append(out, timing)
	if a.AreaDesc != "" {
		out = append(out, wrapPrefixed(o, "Area: "+a.AreaDesc, wrapW)...)
	}
	if a.Description != "" {
		out = append(out, "")
		out = append(out, wrapPrefixed(o, render.Plain(a.Description), wrapW)...)
	}
	if a.Instruction != "" {
		out = append(out, "")
		out = append(out, wrapPrefixed(o, "Instructions: "+a.Instruction, wrapW)...)
	}
	// UAT 55: body text (everything below the toned title) reads white for
	// contrast - advisories and alerts earn it.
	for i := 1; i < len(out); i++ {
		if out[i] != "" {
			out[i] = render.Tint(out[i], render.Tok(render.AlertModalText))
		}
	}
	return append(out, "")
}

// wrapPrefixed wraps prose to the modal width with the 2-col text inset.
func wrapPrefixed(_ render.Opts, text string, w int) []string {
	wrapped := render.WrapText(text, w)
	for i, l := range wrapped {
		wrapped[i] = "  " + l
	}
	return wrapped
}

// alertArea renders the alert module at a FIXED height (UAT 5.2: the area
// stays reserved-but-blank when the focused location has no alert, so the
// UI never jumps). Borderless background block per UAT 5.4: warning-grade
// red-on-red-tint, advisory-grade yellow-on-yellow-tint; the focused
// location's name sits in the title line (UAT 5.3).
// Alert module content: title + blank + THREE body lines (UAT 15.2a) +
// blank + pager = 7; the seam adds bg padding when the tone is visible
// (UAT 19.1 global inset policy).
const alertContentLines = 7

const alertBodyLines = 3

func (d Dashboard) alertArea(fl frameLayout) string {
	o := fl.o
	sel := d.selectedLocation()
	if sel == nil || len(sel.Alerts) == 0 {
		// Reserve the module's CURRENT height (tone visibility included) so
		// the layout never jumps when alerts appear (UAT 5.2 / 19.1).
		return strings.Repeat("\n", fl.alertH-1)
	}
	a := sel.Alerts[d.alertIdx%max(1, len(sel.Alerts))]
	fg, bg := render.AlertBlockTone(a.Event, a.Severity)
	mw := o.ModuleInnerWidth(bg)
	if fl.compact {
		return o.Module([]string{d.alertCompactLine(o, sel, a, mw)}, fg, bg) // UAT 34
	}
	title := o.Glyphs().Alert + " " + strings.ToUpper(a.Event) + " · " + sel.Label
	// UAT 15.2: wrap, never truncate; fixed 3-line body area (15.2a); the
	// MESSAGE alone carries a 4-col inset each side (UAT 19.1).
	body := render.WrapText(fmt.Sprintf("[%s] %s", a.Severity, a.Headline), mw-8)
	if len(body) > alertBodyLines {
		body = body[:alertBodyLines]
	}
	for i := len(body); i < alertBodyLines; i++ { // counter form (P10-02)
		body = append(body, "")
	}
	lines := []string{title, ""}
	for _, l := range body {
		lines = append(lines, "    "+l)
	}
	// UAT 21.1: paging chips mute when the press would do nothing.
	controls := o.KeyCap("A") + " Alert Details   " +
		o.KeyCapIf("←", d.alertIdx > 0) + " Previous  " +
		o.KeyCapIf("→", d.alertIdx < len(sel.Alerts)-1) + " Next"
	lines = append(lines, "", render.PadBetween(fmt.Sprintf("%02d / %02d Alerts", d.alertIdx+1, len(sel.Alerts)), controls, mw))
	return o.Module(lines, fg, bg)
}

// alertCompactLine is the one-row alert module (UAT 34):
// "nn/nn  ⚠ EVENT · Label    [sev] headline...   [A] Alert Details [←] Previous [→] Next".
func (d Dashboard) alertCompactLine(o render.Opts, sel *snapshot.Location, a snapshot.Alert, mw int) string {
	n := len(sel.Alerts)
	head := fmt.Sprintf("%02d/%02d  %s %s · %s", d.alertIdx%n+1, n, o.Glyphs().Alert, strings.ToUpper(a.Event), sel.Label)
	controls := o.KeyCap("A") + " Alert Details  " + o.KeyCapIf("←", d.alertIdx > 0) + " Previous  " + o.KeyCapIf("→", d.alertIdx < n-1) + " Next"
	body := fmt.Sprintf("[%s] %s", a.Severity, a.Headline)
	room := mw - render.Width(head) - render.Width(controls) - 7 // 4-col gap + 3-col gap
	if room < 8 {
		body = ""
	} else if render.Width(body) > room {
		body = truncateTo(body, room-3) + "..."
	}
	// Progressive degrade (UAT 35): body, then chip labels, then the title
	// itself - the row never exceeds the module width.
	if render.Width(head)+render.Width(controls)+3 > mw {
		controls = o.KeyCap("A") + " " + o.KeyCapIf("←", d.alertIdx > 0) + " " + o.KeyCapIf("→", d.alertIdx < n-1)
	}
	if over := render.Width(head) + render.Width(controls) + 3 - mw; over > 0 {
		head = truncateTo(head, max(8, render.Width(head)-over-1)) + "…"
	}
	return render.PadBetween(head+"    "+body, controls, mw)
}
