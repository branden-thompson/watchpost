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
func truncateTo(s string, n int) string { return render.TruncateCells(s, n) } // by display CELLS: a wide-rune name must not overflow the row (R5-C-10)

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
	title := fmt.Sprintf("ALERT %d / %d · %s", d.alertIdx%len(sel.Alerts)+1, len(sel.Alerts), sel.Label) // the page as shown (R5-C-09)
	return d.floatModalToned(o, d.modalWidth(), title, d.alertDetailLines(), fg, alertModalBG(a))
}

// alertModalBG is the tint an alert sits on — the modal's warning red or
// advisory yellow — in [A] and in the dashboard module alike.
func alertModalBG(a snapshot.Alert) string {
	if render.AlertIsWarning(a.Event, a.Severity) {
		return render.Tok(render.AlertModalWarnBG)
	}
	return render.Tok(render.AlertModalAdvBG)
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
	lines = append(lines, alertRecordLines(o, sel.Alerts[idx], wrapW, alertClock(sel))...)
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
func alertRecordLines(o render.Opts, a snapshot.Alert, wrapW int, in *time.Location) []string {
	tone := modalAlertTone(a)                      // UAT 28.3/28.4 modal text tones
	head := strings.ToUpper(render.Plain(a.Event)) // provider text never addresses the terminal (S-F6) — EVERY field, not just the title (0.13.0 P4-1)
	meta := fmt.Sprintf("[%s · %s · %s]", render.Plain(a.Severity), render.Plain(a.Urgency), render.Plain(a.Certainty))
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
		out = append(out, wrapPrefixed(o, "Area: "+render.Plain(a.AreaDesc), wrapW)...)
	}
	if a.Description != "" {
		out = append(out, "")
		out = append(out, wrapPrefixed(o, render.Plain(a.Description), wrapW)...)
	}
	if a.Instruction != "" {
		out = append(out, "")
		out = append(out, wrapPrefixed(o, "Instructions: "+render.Plain(a.Instruction), wrapW)...)
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

// The alert module (HUM LEAD UAT 2026-08-28 facelift): ONE row in a heavy
// box on the Alert Details modal's tint — the warning red or the advisory
// yellow — "02/02  ⚠ FLOOD ADVISORY - Temecula, CA  • Issued: 08/26 8:00 AM
// • Expires: 12/31 12:59 PM" with the paging chips at the right. The body
// lives behind [A] ("dive in for details"); the module reclaims the rows.
// Without an alert the box still stands, muted, so the layout never jumps
// (UAT 5.2 / 19.1). The compact layout keeps its one row (UAT 34) — the
// same tinted line without the rules, so the short terminal's window holds.
func (d Dashboard) alertArea(fl frameLayout) string {
	o := fl.o
	sel := d.selectedLocation()
	fg, bg := render.ModalTone(d.darkBG)
	label := "no location"
	if sel != nil {
		label = sel.Label
	}
	var line string
	if sel != nil && len(sel.Alerts) > 0 {
		a := sel.Alerts[d.alertIdx%len(sel.Alerts)]
		bg = alertModalBG(a)
		if fl.compact {
			return o.Block(d.alertLine(o, sel, a, o.Width), fg, bg)
		}
		line = d.alertLine(o, sel, a, o.BoxInnerWidth())
	} else {
		line = render.Tint("No active alerts · "+label, render.Tok(render.TextBase))
		if fl.compact {
			return o.Block(line, fg, bg)
		}
	}
	return o.Box([]string{line}, fg, bg)
}

// alertLine is the module's row: count · TITLE (the glyph and EVENT bold
// in the alert's modal tone — the same dress as the record's title inside
// [A] — the place bold white) · the severity token when colour is off or
// under --ascii (R-12a: the class in text, not tint alone — round 4, B-04)
// · issued · expires, the chips right-aligned. Progressive degrade (UAT 35):
// expires, then issued, then the chip labels, then the title itself
// shortens — the row never exceeds the module width. UAT 21.1: paging
// chips mute when the press would do nothing.
func (d Dashboard) alertLine(o render.Opts, sel *snapshot.Location, a snapshot.Alert, mw int) string {
	n := len(sel.Alerts)
	idx := d.alertIdx % n // the page as shown: the list may have shrunk under the raw index (R5-C-09)
	count := fmt.Sprintf("%02d/%02d  ", idx+1, n)
	event := o.Glyphs().Alert + " " + strings.ToUpper(render.PlainLine(a.Event)) // every field crosses the boundary (NFR-6)
	title := event + " - " + sel.Label
	controls := o.KeyCap("A") + " Details   " + o.KeyCapIf("←", idx > 0) + " Previous   " + o.KeyCapIf("→", idx < n-1) + " Next"
	in := alertClock(sel)
	stamp := func(t time.Time) string { return t.In(in).Format("01/02 3:04 PM") }
	issued, expires := "", ""
	if !a.Sent.IsZero() {
		issued = "  • Issued: " + stamp(a.Sent)
	} else if !a.Effective.IsZero() {
		issued = "  • Issued: " + stamp(a.Effective)
	}
	if end := a.Expires; a.Ends != nil || !end.IsZero() {
		if a.Ends != nil {
			end = *a.Ends
		}
		expires = " • Expires: " + stamp(end)
	}
	class := ""
	if !render.ColorOn() || o.ASCII {
		class = "  [" + render.PlainLine(a.Severity) + "]"
	}
	fixed := render.Width(count) + render.Width(class) + 3
	stamps := render.FirstFit(mw-fixed-render.Width(title)-render.Width(controls), issued+expires, issued, "")
	if fixed+render.Width(title)+render.Width(controls) > mw {
		controls = o.KeyCap("A") + " " + o.KeyCapIf("←", idx > 0) + " " + o.KeyCapIf("→", idx < n-1)
	}
	if over := fixed + render.Width(title) + render.Width(controls) - mw; over > 0 {
		title = truncateTo(title, max(8, render.Width(title)-over-1)) + "…"
	}
	// The dress goes on after the ladder: the event in the modal's tone,
	// the place bold white (the modal title's tone) — a shortened title
	// keeps whatever the ellipsis left of each.
	styled := render.TintRaw(title, "1;"+modalAlertTone(a))
	if ev, place, ok := strings.Cut(title, " - "); ok {
		styled = render.TintRaw(ev, "1;"+modalAlertTone(a)) + " - " + render.TintRaw(place, render.Tok(render.ModalTitle))
	}
	return render.PadBetween(count+styled+class+stamps, controls, mw)
}
