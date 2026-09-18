package app

// bedfence.go — how many relays a bed fence reaches, and what to say about it
// (D-77).
//
// THE HUM LEAD ASKED FOR THE FEEDBACK BY NAME (2026-09-10): "we should provide
// the user feedback on how many relays they get based on their radius … and if
// the user increases / decreases we call out the various risks associated with
// it."
//
// ONE OWNER, SO TWO SURFACES CANNOT DISAGREE. Settings will show it beside the
// control and the console shows it beside the bed; a second copy of "how many,
// and is that enough" is a second answer to a question the operator is using to
// make a decision.
//
// AND IT IS ARCHITECTED FOR A SETTINGS ROW THAT DOES NOT EXIST YET — "even if we
// immediately don't make this setting visible in settings, it should be
// architected this way so it is literally a 'flip of a switch'". The number and
// the sentence are computed here, by a pure function of the fence; the row that
// shows them is the only thing still missing.

import (
	"strconv"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
)

// bedReach is what a bed fence gets the operator: how many relays, and whether
// that is enough to be a choice.
type bedReach struct {
	Relays int

	// Advice is what to tell them about it — empty when the fence is
	// unremarkable, because a warning on every value is a warning about nothing.
	Advice string
}

// bedReachFor counts the transmitters a fence reaches and says what that means.
//
// THE NUMBERS IN THE THRESHOLDS ARE MEASURED, not chosen. Around Bonsall a
// 25-mile fence reaches ONE transmitter and 100 reaches eight; Lone Pine reaches
// NONE inside fifty miles. So "none" and "one" are the two states worth naming —
// one is a station with no bed at all, the other is a selector with nothing to
// select — and a wide fence is worth naming for the opposite reason.
func bedReachFor(table *stream.Table, lat, lon, radiusMi float64) bedReach {
	n := len(table.Within(lat, lon, radiusMi))
	return bedReach{Relays: n, Advice: bedAdvice(n, radiusMi)}
}

// bedAdvice is the sentence, separated from the counting so the wording can be
// asserted without a table and the counting without a sentence.
func bedAdvice(relays int, radiusMi float64) string {
	switch {
	case relays == 0:
		return "no relay reaches this far — the bed has nothing to carry; widen the search"
	case relays == 1:
		return "one relay in range — there is nothing to switch between; widen the search for a choice"
	case radiusMi >= bedFenceFarMi:
		// THE RISK AT THE OTHER END, and it is the objection that made the bed
		// the station's business at all: a relay this far out is broadcasting a
		// forecast for a region the station's own listeners are not in.
		return "relays this far out cover a different forecast area than your listeners"
	}
	return ""
}

// bedFenceFarMi is where a relay stops being local to the station.
//
// A HUNDRED AND TWENTY-FIVE, which is above the ruled DEFAULT of 100: the
// default must not arrive wearing a warning, or the warning is what the operator
// learns to ignore.
const bedFenceFarMi = 125

// bedReachLine is the one-line form both surfaces show: the count, and the
// advice when there is any.
func (b bedReach) Line() string {
	s := strconv.Itoa(b.Relays) + " relay"
	if b.Relays != 1 {
		s += "s"
	}
	s += " in range"
	if b.Advice != "" {
		s += " — " + b.Advice
	}
	return s
}
