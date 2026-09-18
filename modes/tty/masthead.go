package tty

// masthead.go — the framed header BOTH surfaces draw (D-59, D-56).
//
// THE HUM LEAD ASKED WHY THE CONSOLE'S MASTHEAD WAS DIFFERENT FROM THE
// OBSERVER'S. It should not have been: I had deviated from the reference —
// dropping the version from the title, replacing the `Updated:` stamp with an
// invented "ON AIR / STANDBY", and leaving out the API summary. The reference
// draws all three, on both surfaces.
//
// SO THE LADDERS LIVE HERE, ONCE. Everything about the masthead that is the same
// on both surfaces is one function called twice, which is "one canonical way to
// do a thing" applied to the thing that prompted the question. What genuinely
// differs is the EDITION WORD and, on the console, one extra row naming the
// station — and those arrive as arguments rather than as a second header.

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// mastheadTitle is the wordmark, the edition and the version, at the widest
// form the rule can carry.
//
// THE ORDER IS RULED AND IT IS THE OBSERVER'S: the version leaves first, then
// the edition, and the wordmark is the last thing to go — "a masthead that
// cannot say what the app is has stopped being a masthead."
//
// Each rung is built only when the one above it did not fit, because the bare
// wordmark costs a per-rune gradient pass for a form only a very narrow terminal
// ever shows.
func mastheadTitle(edition, version string, rule int) string {
	full := render.Wordmark(edition)
	if title := full + "  v" + version; render.Width(title)+4 <= rule {
		return title
	}
	if render.Width(full)+4 <= rule {
		return full
	}
	return render.Wordmark("")
}

// mastheadStamp is when the data last came home, in the widest form that fits
// beside the title — and its TONE says whether that is recent.
//
// GREY BEFORE THE FIRST DATA, green while it is fresh, yellow once no fetch has
// succeeded for `staleAfter`. The colour is the only part of the masthead that
// is not a word, and it is the part an operator reads without looking.
func mastheadStamp(o render.Opts, title string, snap *snapshot.Snapshot, now time.Time, rule int) string {
	forms, tone := []string{"awaiting first data..."}, render.Tok(render.TextBase)
	if snap != nil {
		at := dataAsOf(snap)
		full := "Updated: " + o.Clock.Stamp(at.Local())
		short := o.Clock.TimeSec(at.Local())
		forms = []string{full + " (" + agoWords(now.Sub(at)) + ")", full, "Updated: " + short, short}
		tone = render.Tok(render.ProviderOK)
		if now.Sub(at) > staleAfter {
			tone = render.Tok(render.AlertLabel)
		}
	}
	for _, form := range forms {
		if render.Width(title)+4+render.Width(form)+5 <= rule {
			return render.Tint(form, tone)
		}
	}
	return ""
}
