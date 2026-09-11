package tty

// broadcaster_header.go — the console's masthead (D-59).
//
// IT REUSES THE OBSERVER'S OWN BOX (D-56). `render.BoxTitled` already draws
// this exact shape — `┌── TITLE ──…── STAMP ──┐` — and the Dashboard already
// carries the responsive LADDER: progressively shorter forms, each built only
// when the wider one did not fit. A second box drawer here would be a second
// place for the frame to drift, and the two would drift on the day one of them
// learned a new breakpoint.
//
// THE EDITION ARRIVES AS A WORD, NOT A RENAME, which `sgr.go` has said since
// 2026-08-30: "Broadcaster is the station-running dashboard a later version
// brings, and it will pass its own word through here."

import (
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// header is the masthead block: the framed title row, the controls, and the
// station's own identity.
func (b Broadcaster) header(o render.Opts) string {
	rule := o.BoxRuleWidth()
	title := mastheadTitle(render.EditionBroadcaster, b.version, rule)
	stamp := mastheadStamp(o, title, b.snap, b.clock(), rule)
	return o.BoxTitled(b.headerRows(o), title, stamp, "", "")
}

// headerRows is what rides inside the masthead: the controls the operator has,
// and nothing else.
//
// THE STATION'S IDENTITY LEFT AT D-71 (HUM LEAD, UAT 2026-09-10): "we're going
// to take the station center out of the masthead — this should now make the
// Observer/Broadcaster masthead nearly identical minus the Observer/Broadcaster
// [word]." It is a fact about the STATION, and the station has a section of its
// own; the masthead is what the two surfaces share.
func (b Broadcaster) headerRows(o render.Opts) []string {
	return []string{b.controlRow(o)}
}

// controlRow names the keys, with the surface swap among them.
//
// A LADDER AGAIN, and the last rung keeps the two controls that are ONLY named
// here: the swap back to Observer, and quit. A narrow console that named no key
// at all would strand an operator who arrived by keyboard.
func (b Broadcaster) controlRow(o render.Opts) string {
	inner := o.BoxInnerWidth()
	// THE API SUMMARY RIDES AT THE RIGHT, which the reference draws and I had
	// left out. It is the Observer's own count, from the same snapshot, so the
	// two surfaces cannot report different provider health.
	api := apiSummaryOf(o, b.snap)
	// EVERY KEY IS A CHIP, THROUGH `o.Controls` — the Observer's own control-row
	// builder, and the reason this row is not built from string literals any
	// more. It was, and the HUM LEAD saw the result in UAT (2026-09-10): "chips
	// don't render their bkg ... tells me something about coloring and tokens
	// are broken in broadcaster ui". Nothing was broken. The console had simply
	// TYPED the keys as text, so no chip existed to paint.
	//
	// `[ X ]` IN THE MOCK IS A CHIP CONTROL, NOT BRACKETS (HUM LEAD): "the
	// brackets indicate a chip control ... that's also true throughout the
	// layout". `KeyCap` is the one thing that knows that — the painted cap in
	// colour, the literal `[X]` without it, `[x]` under --ascii.
	for _, form := range [][]render.Control{
		{render.Ctl("s", "Settings"), render.Ctl("a", "About"), render.Ctl("S", "Status"),
			render.Ctl("ctrl+o", "Observer"), render.Ctl("?", "Help"), render.Ctl("q", "Quit")},
		{render.Ctl("s", "Settings"), render.Ctl("ctrl+o", "Observer"),
			render.Ctl("?", "Help"), render.Ctl("q", "Quit")},
		{render.Ctl("ctrl+o", "Observer"), render.Ctl("q", "Quit")},
	} {
		row := o.Controls("  ", form...)
		if render.Width(row)+2+render.Width(api) <= inner {
			return render.PadBetween(row, api, inner)
		}
	}
	// The bare floor still names the two controls that appear ONLY here: the way
	// back to Observer, and the way out.
	return render.PadTo(render.TruncateCells(o.Controls("  ",
		render.Ctl("ctrl+o", "Observer"), render.Ctl("q", "Quit")), inner), inner)
}

// transmitterRow is where the station broadcasts FROM, and how far it reaches.
//
// IT READS THE PUBLISHED SETTINGS (D-72). Both facts were placeholder constants
// until the station got settings of its own — `BROADCAST LOCATION : Bonsall, CA`
// and `SERVICE RADIUS: 100 Miles` were the reference mock's values, hard-coded,
// on a console whose whole job is to say what the station is actually doing.
//
// THE COORDINATES ARE REAL HERE AND A PLACEHOLDER IN THE DOCUMENTS (F-66). The
// repository is public and the mock renders `<lat>, <lon>`; the operator's own
// screen renders the operator's own tower.
func (b Broadcaster) transmitterRow(inner int) string {
	where, reach := b.area.Transmitter.Label, ""
	if where == "" {
		// NO EPICENTRE IS A REAL STATE AND IT SAYS SO. A station with no
		// transmitter has no region, an empty pool and ten shimmering slots —
		// and the operator is owed the reason, not just the symptom.
		where = bcNoTransmitter
	}
	gps := "TOWER GPS:  " + bcNoCoordinates
	if t := b.area.Transmitter; t.Lat != 0 || t.Lon != 0 {
		gps = "TOWER GPS:  " + strconv.FormatFloat(t.Lat, 'f', 6, 64) + ", " + strconv.FormatFloat(t.Lon, 'f', 6, 64)
	}
	if b.area.RadiusMi > 0 {
		reach = "SERVICE RADIUS: " + strconv.FormatFloat(b.area.RadiusMi, 'f', -1, 64) + " Miles"
	}
	// Widest first: all three, then the two that say WHERE and HOW FAR, then
	// the location alone. The GPS is the first to go because it is the one
	// field a listener never needs and an operator can find elsewhere.
	for _, form := range [][]string{{where, gps, reach}, {where, reach}, {where}} {
		row := strings.Join(nonEmpty(form), "    ")
		if render.Width(row) <= inner {
			return render.PadTo(row, inner)
		}
	}
	return render.PadTo(render.TruncateCells(where, inner), inner)
}

// nonEmpty drops the fields the station has not set, so a ladder rung does not
// render as a run of separators around nothing.
func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in { // bounded by the form (P10-02)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

const (
	// bcNoTransmitter and bcNoCoordinates are what the row says before the
	// station has an epicentre (D-72). They replaced `bcPlaceholderLocation` and
	// `bcPlaceholderRadius`, which were the reference mock's values HARD-CODED —
	// a console that reported a station it had not been told about.
	bcNoTransmitter = "(no transmitter set)"
	bcNoCoordinates = "--.------, ---.------"
)
