package app

import (
	"slices"
	"sync"
	"sync/atomic"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// One owner for the things the two producing decks — the radio and the ticker —
// were each stating for themselves.
//
// THIS FILE EXISTS BECAUSE OF ISSUE #7. That bug was a name-to-install lookup
// written out four times, and the fourth was written by copying the third with
// the comment "find-only, exactly as the deck's is" — faithfully, defect and
// all. Nothing here is currently wrong. It is here because a policy stated in
// two places has already proved it will be changed in one.

// clockFrom is the listener's clock, twelve-hour when nothing has set one.
//
// The default is a POLICY, and radioDeck and tickerDeck each held their own
// copy of it. They agree today; the failure mode is that they stop, and the
// radio and the tape then disagree about what time it is on one screen.
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
// Both producers had their own copy of it.
func tellUnder(mu *sync.Mutex, emit *func(lineup.Event), ev lineup.Event) {
	mu.Lock()
	fn := *emit
	mu.Unlock()
	if fn != nil {
		fn(ev)
	}
}

// discoveredIn is the closed allowlist both hosts apply: the curated list until
// `say -v ?` has answered, the intersection afterwards, and the macOS sentinel
// always present.
//
// radioDeck and hostFacts each wrote this out. hostfacts.go's own doc comment
// says why that is dangerous — "two implementations of which voices does this
// host have is how a report and a screen start disagreeing about the same
// machine, so the deck and the report share this one" — and then it did not
// share it. Installed, the third method of the same interface, is how issue #7
// spread.
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
