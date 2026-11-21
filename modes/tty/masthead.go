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
	"fmt"
	"strconv"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// MastheadTitle is the wordmark, the edition and the version, at the widest
// form the rule can carry.
//
// THE ORDER IS RULED AND IT IS THE OBSERVER'S: the version leaves first, then
// the edition, and the wordmark is the last thing to go — "a masthead that
// cannot say what the app is has stopped being a masthead."
//
// Each rung is built only when the one above it did not fit, because the bare
// wordmark costs a per-rune gradient pass for a form only a very narrow terminal
// ever shows.
func MastheadTitle(edition, version string, rule int) string {
	full := render.Wordmark(edition)
	if title := full + "  v" + version; render.Width(title)+4 <= rule {
		return title
	}
	if render.Width(full)+4 <= rule {
		return full
	}
	return render.Wordmark("")
}

// MastheadStamp is when the data last came home, in the widest form that fits
// beside the title — and its TONE says whether that is recent.
//
// GREY BEFORE THE FIRST DATA, green while it is fresh, yellow once no fetch has
// succeeded for `staleAfter`. The colour is the only part of the masthead that
// is not a word, and it is the part an operator reads without looking.
func MastheadStamp(o render.Opts, title string, snap *snapshot.Snapshot, now time.Time, rule int) string {
	forms, tone := []string{"awaiting first data..."}, render.Tok(render.TextBase)
	if snap != nil {
		at := DataAsOf(snap)
		full := "Updated: " + o.Clock.Stamp(at.Local())
		short := o.Clock.TimeSec(at.Local())
		forms = []string{full + " (" + AgoWords(now.Sub(at)) + ")", full, "Updated: " + short, short}
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

// MastheadKeyOf derives the key for a masthead drawn at now.
func MastheadKeyOf(o render.Opts, edition, version string, snap *snapshot.Snapshot, now time.Time) mastheadKey {
	o.Frame = 0
	k := mastheadKey{o: o, edition: edition, version: version, counts: countProviders(snap),
		theme: render.ThemeGeneration(), colour: render.ColorOn()}
	if snap != nil {
		at := DataAsOf(snap)
		k.hasData, k.asOf, k.ago, k.stale = true, at.UnixNano(), AgoWords(now.Sub(at)), now.Sub(at) > staleAfter
	}
	return k
}

// MastheadSlot is one surface's masthead memo, a single entry: the masthead
// is drawn on every frame and changes when its data or its age words do, so
// the frames between redraw it from here (0.19.5 PF-7, D-22). A nil slot (a
// model without a memo) builds every time.
type MastheadSlot struct {
	ok  bool
	key mastheadKey
	out string
}

// Get is the masthead for k, built only when k is not the one held.
func (s *MastheadSlot) Get(k mastheadKey, build func() string) string {
	if s == nil {
		return build()
	}
	if !s.ok || s.key != k {
		s.ok, s.key, s.out = true, k, build()
	}
	return s.out
}

// staleAfter is how long the stamp stays green after the last successful
// fetch (UAT 2026-08-27: "the API has not successfully refreshed in > 5 minutes").
const staleAfter = 5 * time.Minute

// AgoWords is the stamp's age in words: "Just Now", "1 Minute Ago",
// "12 Minutes Ago", "3 Hours Ago".
func AgoWords(age time.Duration) string {
	switch {
	case age < time.Minute:
		return "Just Now"
	case age < time.Hour:
		m := int(age / time.Minute)
		return fmt.Sprintf("%d Minute%s Ago", m, Plural(m))
	}
	h := int(age / time.Hour)
	return fmt.Sprintf("%d Hour%s Ago", h, Plural(h))
}

// APISummaryOf counts the providers' health for the masthead — the ONE owner,
// called by both surfaces (D-59). Two counts of one snapshot could disagree, and
// the number is a health claim.
func APISummaryOf(o render.Opts, snap *snapshot.Snapshot) string {
	c := countProviders(snap)
	ok, stale, down, total := c.ok, c.stale, c.down, c.total
	g := o.Glyphs()
	gOK, gStale, gDown := g.OK, g.Alert, g.Fail // one owner for every mark (R5-C-13)
	if o.ASCII {
		gOK, gStale, gDown = "OK", "!!", "XX"
	}
	return fmt.Sprintf("API: %s %s%d %s%d /%3d",
		render.Tint(gOK+strconv.Itoa(ok), render.Tok(render.ProviderOK)),
		render.Tint(gStale, render.Tok(render.AlertLabel)), stale,
		render.Tint(gDown, render.Tok(render.ProviderDown)), down, total)
}

// providerCounts are the providers counted by health, as APISummaryOf says them.
type providerCounts struct{ ok, stale, down, total int }

// countProviders counts a snapshot's providers by health: the summary's numbers,
// and the masthead memo's key for them.
func countProviders(snap *snapshot.Snapshot) providerCounts {
	var c providerCounts
	for _, p := range ProvidersOf(snap) { // bounded by the providers (P10-02)
		switch {
		case p.Status == snapshot.ProviderOff:
			continue
		case p.Status == snapshot.ProviderOK && p.FetchedAt.IsZero():
			// registered, not yet answered (the first frames): in the total, not yet ✔ (REVIEW C5)
		case p.Status == snapshot.ProviderOK:
			c.ok++
		case p.FetchedAt.IsZero():
			c.down++
		default:
			c.stale++
		}
		c.total++
	}
	return c
}

// DataAsOf is the header's "Last Updated" time (red-team 0.9.0 F7): the
// newest successful provider fetch — not the publish time, which keeps
// ticking while every row is stale last-good data offline.
func DataAsOf(sn *snapshot.Snapshot) time.Time {
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

// ProvidersOf guards the nil snapshot for the status view.
func ProvidersOf(sn *snapshot.Snapshot) []snapshot.ProviderStatus {
	if sn == nil {
		return nil
	}
	return sn.Providers
}
