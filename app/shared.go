package app

import (
	"slices"
	"sync"
	"sync/atomic"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// One owner for the things the two producing decks — the radio and the ticker —
// would otherwise each state for themselves.
//
// A POLICY STATED IN TWO PLACES GETS CHANGED IN ONE, and a copy carries its
// original's defects with it (issue #7).

// clockFrom is the listener's clock, twelve-hour when nothing has set one.
//
// The default is a POLICY, held here once for radioDeck and tickerDeck. Two
// copies could stop agreeing, and the radio and the tape would then disagree
// about what time it is on one screen.
func clockFrom(pref *atomic.Int32) render.Clock {
	if pref == nil {
		return render.Clock12
	}
	return render.Clock(pref.Load())
}

// tellUnder hands one event to the Director, reading the callback under the
// lock and CALLING IT OUTSIDE — the producer must never hold its own mutex
// while the Director runs.
//
// The field is taken by pointer on purpose: passing the func by value would
// read it without the lock, which is the exact discipline this exists to own.
// Both producers use this one.
func tellUnder(mu *sync.Mutex, emit *func(lineup.Event), ev lineup.Event) {
	mu.Lock()
	fn := *emit
	mu.Unlock()
	if fn != nil {
		fn(ev)
	}
}

// setUnder writes one field under its owner's mutex.
//
// ONE HELPER FOR BOTH CALLERS (D-147): `mastercontrol.unhold` and
// `tickerDeck.setScope` are the same twenty-six nodes — nil guard, lock, one
// assignment, unlock. The `dupes` gate refuses such a pair, and a reason is
// RATIFIED, never self-issued, so they share this rather than carry an
// exemption.
//
// IT SITS BESIDE `tellUnder` BECAUSE IT IS THE OTHER HALF OF ONE DISCIPLINE:
// that one reads a callback under the lock and calls it OUTSIDE; this one
// writes a field under the lock and returns. Both take the field by pointer for
// the same stated reason — passing it by value would touch it without the lock,
// which is the exact discipline these exist to own.
func setUnder[T any](mu *sync.Mutex, dst *T, v T) {
	mu.Lock()
	*dst = v
	mu.Unlock()
}

// discoveredIn is the closed allowlist both hosts apply: the curated list until
// `say -v ?` has answered, the intersection afterwards, and the macOS sentinel
// always present.
//
// radioDeck and hostFacts both apply this one: two implementations of "which
// voices does this host have" is how a report and a screen start disagreeing
// about the same machine (issue #7).
func discoveredIn(discovered []string, name string) bool {
	if name == "" {
		return false
	}
	if name == systemVoice {
		return true // the sentinel is always present (UAT 88)
	}
	if len(discovered) == 0 {
		discovered = macVoices() // `say -v ?` has not answered yet
	}
	return slices.Contains(discovered, name)
}

// defaultVoiceFor is this machine's last resort when even the root did not
// resolve: the macOS sentinel, else the first installed Piper voice, else
// nothing — the one legitimately silent row (AM-19).
func defaultVoiceFor(platform, dir string) string {
	if platform == cast.PlatformDarwin {
		return systemVoice
	}
	if installed := synth.InstalledVoices(dir); len(installed) > 0 {
		return installed[0].Name
	}
	return ""
}

// toneClassOfEvent is the tone class an event sounds, and it is the ONE place
// that decides between the product's words and its category.
//
// cast.Classify reads the words, which is right for the severity family. It is
// wrong for the civil-emergency family: "Evacuation Immediate" contains no
// "warning", "watch" or "advisory", so it falls through to Classify's loud
// default — and that fall-through is #18.
//
// NARROW ON PURPOSE. The category wins only for the products whose words
// provably cannot carry it, which is exactly globalfeed's civil-emergency
// table. Preferring the category everywhere would break advisories: LaneOf
// has no Advisory arm and defaults to Warnings, so a Small Craft Advisory would
// sound like a warning.
func toneClassOfEvent(e globalfeed.Event) cast.Class {
	if c, ok := globalfeed.CivilEmergencyCategory(e.Type); ok {
		if cls, ok := cast.ClassFor(c); ok {
			return cls
		}
	}
	return cast.Classify(e.Type)
}
