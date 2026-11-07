package tty

// masthead.go — the framed header BOTH surfaces draw (D-59, D-56).
//
// THE CONSOLE'S MASTHEAD IS THE OBSERVER'S, as the reference draws it: the
// version in the title, the `Updated:` stamp, and the API summary, on both
// surfaces.
//
// SO THE LADDERS LIVE HERE, ONCE. Everything about the masthead that is the same
// on both surfaces is one function called twice, which is "one canonical way to
// do a thing". What genuinely
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

// mastheadKey is everything a masthead is drawn from: the options (all but the
// animation phase, which no masthead reads), the edition and its version, the
// data's as-of instant, the providers' counts the API summary gives, the age
// words and staleness the stamp says, and the theme and colour switch every
// tint reads. The controls inside the box are fixed for each edition.
type mastheadKey struct {
	o                render.Opts
	edition, version string
	asOf             int64
	hasData          bool
	counts           providerCounts
	ago              string
	stale            bool
	theme            uint64
	colour           bool
}

// mastheadKeyOf derives the key for a masthead drawn at now.
func mastheadKeyOf(o render.Opts, edition, version string, snap *snapshot.Snapshot, now time.Time) mastheadKey {
	o.Frame = 0
	k := mastheadKey{o: o, edition: edition, version: version, counts: countProviders(snap),
		theme: render.ThemeGeneration(), colour: render.ColorOn()}
	if snap != nil {
		at := dataAsOf(snap)
		k.hasData, k.asOf, k.ago, k.stale = true, at.UnixNano(), agoWords(now.Sub(at)), now.Sub(at) > staleAfter
	}
	return k
}

// mastheadSlot is one surface's masthead memo, a single entry: the masthead
// is drawn on every frame and changes when its data or its age words do, so
// the frames between redraw it from here (0.19.5 PF-7, D-22). A nil slot (a
// model without a memo) builds every time.
type mastheadSlot struct {
	ok  bool
	key mastheadKey
	out string
}

// get is the masthead for k, built only when k is not the one held.
func (s *mastheadSlot) get(k mastheadKey, build func() string) string {
	if s == nil {
		return build()
	}
	if !s.ok || s.key != k {
		s.ok, s.key, s.out = true, k, build()
	}
	return s.out
}
