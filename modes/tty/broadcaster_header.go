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
	full := render.Wordmark(render.EditionBroadcaster)
	rule := o.BoxRuleWidth()

	// THE LADDER, and its ORDER is the Observer's: the version leaves first,
	// then the edition, and the wordmark is the last thing to go — "a masthead
	// that cannot say what the app is has stopped being a masthead."
	//
	// Each rung is built only when the one above it did not fit, because the
	// bare wordmark costs a per-rune gradient pass for a form only a very narrow
	// terminal ever shows.
	title := full
	switch {
	case render.Width(full)+4 <= rule:
	default:
		title = render.Wordmark("")
	}
	// The stamp is what the station IS ON AIR AS, not a clock: the console has
	// its own STATION line for state, and repeating it here would be two
	// carriers of one fact.
	stamp := ""
	for _, form := range []string{"ON AIR / STANDBY", "ON AIR"} {
		if render.Width(title)+4+render.Width(form)+5 <= rule {
			stamp = form
			break
		}
	}
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
	for _, form := range [][]string{
		{"s  Settings", "a  About", "S  Status", "ctrl+o  Observer", "?  Help", "q  Quit"},
		{"s  Settings", "ctrl+o  Observer", "?  Help", "q  Quit"},
		{"ctrl+o  Observer", "q  Quit"},
	} {
		row := strings.Join(form, "   ")
		if render.Width(row) <= inner {
			return render.PadTo(row, inner)
		}
	}
	return render.PadTo(render.TruncateCells("ctrl+o  Observer", inner), inner)
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
