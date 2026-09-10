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
// and the station's identity.
func (b Broadcaster) headerRows(o render.Opts) []string {
	return []string{b.controlRow(o), b.identityRow(o)}
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

// identityRow is where the station broadcasts FROM, and how far it reaches.
//
// THE COORDINATES ARE A PLACEHOLDER AND STAY ONE (F-66, CLOSED). The repository
// is public; the reference mock renders `TOWER GPS: <lat>, <lon>` and so does
// this. Real coordinates arrive, if ever, behind a ruling of their own.
func (b Broadcaster) identityRow(o render.Opts) string {
	inner := o.BoxInnerWidth()
	const (
		where = "BROADCAST LOCATION : " + bcPlaceholderLocation
		gps   = "TOWER GPS:  <lat>, <lon>"
		reach = "SERVICE RADIUS: " + bcPlaceholderRadius
	)
	// Widest first: all three, then the two that say WHERE and HOW FAR, then
	// the location alone. The GPS row is the first to go because it is the one
	// field a listener never needs and an operator can find elsewhere.
	for _, form := range [][]string{{where, gps, reach}, {where, reach}, {where}} {
		row := strings.Join(form, "    ")
		if render.Width(row) <= inner {
			return render.PadTo(row, inner)
		}
	}
	return render.PadTo(render.TruncateCells(where, inner), inner)
}

const (
	// bcPlaceholderLocation and bcPlaceholderRadius stand in until the station's
	// own settings reach the console. They are the reference mock's values, so
	// what is drawn is what was designed rather than something invented here.
	bcPlaceholderLocation = "Bonsall, CA"
	bcPlaceholderRadius   = "100 Miles"
)
