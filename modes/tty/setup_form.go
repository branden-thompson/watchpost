package tty

// setup_form.go — the three questions that have no group file of their own:
// the DEFAULT LOCATION, the NASA FIRMS key, and the ALERTS - EVENTS radius.
//
// They are the first-run form, which is where this window began before it grew
// the other groups — so they are named for the form rather than for a group
// heading, and the other four groups keep their own files
// (setup_cast/ui/relay/tones.go).
//
// This is CONTENT: the lines each question draws. Its geometry is
// setup_layout.go and its key handling is setup.go.
//
// SPLIT FROM setup.go (2026-09-06), a pure move.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// supportIndent is where a question's SUPPORT text starts: its hint, its
// type-ahead suggestions, its reveal chip and its error, all under the question
// they belong to rather than back at the window's margin.
//
// It is one constant because it was four copies of the same literal, and a
// support line that drifted from its neighbours would read as a rendering bug
// (modularity standard: extract at the second caller). The width lines the
// support text up past the two-cell focus mark and the two-cell group indent
// that a question's own head carries.
const supportIndent = "       "

// setupLocationLines is question 1 of the form.
func (d Dashboard) setupLocationLines(o render.Opts, mark string) []string {
	st := d.setup
	unset := false
	// The mock's wording (OP-2): a label and the value on one line, so the
	// group is narrow enough for two columns. The hint moves to the support
	// line under it, where the other groups keep theirs.
	head := "  " + mark + settingLabel("Default location: ", st.focus == rowLocation)
	// THE APPLICATION'S DEFAULT IS SHOWN, AND SAID TO BE ONE (T4.2, R-4). With
	// no watchlist and nothing chosen, the row used to read "Default location: "
	// and then nothing at all, which tells a listener neither what the station
	// would reason from nor that it is waiting on them. Bonsall is named, and
	// LABELLED as the application's rather than theirs — "shown in Settings as
	// the Default, never used silently".
	switch cur := d.currentDefault(); {
	case st.ref != nil:
		head += render.Plain(st.ref.Label) + " (" + st.ref.Zip + ")"
	case cur != nil:
		head += render.Plain(cur.Label) + " (" + cur.Zip + ")"
	default:
		origin := snapshot.DefaultOrigin()
		head += render.Plain(origin.Label) + " (" + origin.Zip + ")"
		unset = true
	}
	// Questions read white, support grey (UAT 111.5) — so the note that this is
	// the APPLICATION's default and not the listener's goes on the support line,
	// where the group's other explanations live, rather than tinted onto the
	// question beside the value.
	hint := "City Name, \"City, ST\", or Zip"
	if unset {
		hint = "Watchpost's default " + o.Glyphs().Dash + " set your own: " + hint
	}
	lines := []string{head, supportIndent + hint}
	if st.ref == nil || st.focus == rowLocation {
		lines = append(lines, "       Search: "+st.query+o.Glyphs().Cursor)
		// THROUGH THE LIST'S ONE OWNER (D-1). This drew a bare "›" and tinted
		// nothing — the SAME three defects the relay-fault and ctrl+d windows
		// were fixed for, in the most-opened window in the app, and both of
		// those files carry a comment calling theirs "the last hardcoded
		// pointer in the tree" (red team 2026-09-06). It was not: --ascii could
		// not turn this one into ">", and the suggestion under the cursor
		// changed no colour, so focus was carried by a glyph alone.
		//
		// ListMark is two cells whether or not a row is focused, exactly as the
		// literals it replaces were, so the column does not move.
		for i, h := range st.hints { // bounded by the suggestion list (P10-02)
			focused := i == st.idx
			lines = append(lines, supportIndent+o.ListMark(focused)+
				render.ListLabel(render.Plain(h.Label)+" ("+h.Zip+")", focused))
		}
	}
	if st.err != "" && st.focus == rowLocation {
		lines = append(lines, "       "+o.Glyphs().Alert+" "+st.err)
	}
	return lines
}

// setupKeyLines is question 2 of the form: the FIRMS key, with a stored
// key's tail and health when there is one (UAT 111).
// setupKeyLines takes Opts because its marks come from the SET, not from
// literals (red team, 2026-09-08: the ellipsis, the dash, the bullet and the
// warning here all survived --ascii, in a window the scan renders only in a
// state that reaches none of them).
func (d Dashboard) setupKeyLines(o render.Opts, mark string) []string {
	st := d.setup
	hint := ""
	if d.cfg.FIRMSKey != nil {
		hint = d.cfg.FIRMSKey()
	}
	// NO LEADING BLANK: the separator between the two DATA rows belongs to the
	// block that lays them out, not to this row. With it here the row's recorded
	// first line was the blank ABOVE the question, so the scroll aimed one line
	// high and a test that asked "is the focus mark on the line you said?" found
	// it was not.
	var lines []string
	if hint != "" { // UAT 111: a stored key is shown to be there, with how it is doing, and can be replaced
		lines = append(lines, "  "+mark+settingLabel("NASA FIRMS key: stored ("+o.Glyphs().Ellipsis+hint+") "+o.Glyphs().Dash+" ", st.focus == rowFIRMSKey)+d.firmsHealth(o),
			"       Paste new key to replace "+o.Glyphs().Dash+" empty keeps")
	} else {
		lines = append(lines, "  "+mark+settingLabel("NASA FIRMS key: ", st.focus == rowFIRMSKey)+"none (optional)",
			"       Free key: firms.modaps.eosdis.nasa.gov/api/map_key",
			"       Empty = the default data set, no key")
	}
	shown := strings.Repeat(o.Glyphs().Bullet, len([]rune(st.key)))
	if st.reveal {
		shown = st.key
	}
	lines = append(lines, "       Key: "+shown+d.opts().Glyphs().Cursor)
	// Only when there is something to reveal, and naming which way it goes.
	//
	// It reveals what you TYPE. A stored key cannot be shown: the app hands
	// this window only the tail ("…d9a6"), deliberately, so the full secret
	// never reaches a frame. A chip promising to reveal a stored key would be
	// promising something the design refuses to do.
	if st.key != "" {
		label := "Reveal typed key"
		if st.reveal {
			label = "Hide typed key"
		}
		lines = append(lines, supportIndent+d.opts().KeyCap("ctrl+r")+"  "+label)
	}
	if st.err != "" && st.focus == rowFIRMSKey {
		lines = append(lines, "       "+o.Glyphs().Alert+" "+st.err)
	}
	return lines
}

// setupAlertLines is the ALERTS - EVENTS group: All locations, or Within N mi.
// Two ROWS now, not one line with two marks — the keyboard rule walks rows, and
// a row is what the › marks.
func (d Dashboard) setupAlertLines(o render.Opts) []string {
	st := d.setup
	buf := st.radiusMi
	if st.focus == rowEventsWithin {
		buf += o.Glyphs().Cursor // the miles cursor, only while editing a distance
	}
	lines := []string{
		"  " + setupMark(o, st.focus == rowEventsAll) + radioMark(!st.filtered, o.ASCII) + " " + settingLabel("All locations (Default)", st.focus == rowEventsAll),
		"  " + setupMark(o, st.focus == rowEventsWithin) + radioMark(st.filtered, o.ASCII) + " " + settingLabel("Within", st.focus == rowEventsWithin) + " [" + render.PadTo(buf, 4) + "] mi of Default Location",
	}
	if st.err != "" && (st.focus == rowEventsAll || st.focus == rowEventsWithin) {
		lines = append(lines, supportIndent+o.Glyphs().Alert+" "+st.err)
	}
	return lines
}

// radioMark is a radio-button glyph: ● selected / ○ not (or * / o under
// --ascii) — the mark carries the choice without colour (R-12a).
func radioMark(selected, ascii bool) string {
	switch {
	case selected && ascii:
		return "*"
	case selected:
		return "●"
	case ascii:
		return "o"
	default:
		return "○"
	}
}

// firmsHealth words the FIRMS provider's state for the Setup window (UAT
// 111): ✔ working (green), ✘ rejected (red), degraded, off, or not yet
// reported — glyph and colour together (R-12a: the glyph carries it alone).
func (d Dashboard) firmsHealth(o render.Opts) string {
	if d.snap == nil {
		return "no report yet"
	}
	for _, w := range d.snap.Warnings {
		if w.Provider == "firms" && strings.Contains(w.Message, "rejected the MAP_KEY") {
			return render.Tint(d.opts().Glyphs().Fail+" rejected "+d.opts().Glyphs().Dash+" replace it", render.Tok(render.ProviderDown))
		}
	}
	for _, p := range d.snap.Providers {
		if p.ID != "firms" {
			continue
		}
		switch p.Status {
		case snapshot.ProviderOK:
			return render.Tint(o.Glyphs().OK+" working", render.Tok(render.ProviderOK))
		case snapshot.ProviderOff:
			return "not active"
		}
		return render.Tint(d.opts().Glyphs().Fail+" degraded (see [S] Status)", render.Tok(render.ProviderDown))
	}
	return "no report yet"
}
