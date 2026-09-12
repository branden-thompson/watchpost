package app

// bedrelay.go — the relays the station may carry on its bed, and the operator's
// walk through them (D-78).
//
// THE SELECTOR THE REFERENCE HAS DRAWN SINCE THE FIRST WAVE and that was bound
// to nothing: `[ ← ] KIG78 Coachella CA 162.400 MHz · 41mi from TOWER GPS
// [ → ]`. What it walks is the STATION's fence (D-77), never the listener's
// watchlist — which is the HUM LEAD's Lone Pine objection, and the reason the
// bed became the station's business at all.
//
// A SELECTION, NOT A ROTATION. The bed does not advance on a dwell the way the
// monitor's watchlist does: an operator CHOOSES what their station carries
// between reads, and keeps it until they choose again.

import (
	tea "charm.land/bubbletea/v2"

	"strconv"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
)

// bedRelays is what the station's fence reaches, nearest first.
//
// DERIVED ON EVERY ASK, like the pool: it is a pure function of the transmitter,
// the bed's fence and the embedded table, and a stored copy would be a second
// answer that could drift from the settings that produced it.
func (lp *livePipelines) bedRelays() []stream.Near {
	if lp == nil || lp.relayTable == nil {
		return nil
	}
	s := lp.currentStation()
	if s.transmitter.Lat == 0 && s.transmitter.Lon == 0 {
		return nil // no epicentre, no region, no relays to choose between
	}
	return lp.relayTable.Within(s.transmitter.Lat, s.transmitter.Lon, lp.bedFenceMi())
}

// bedFenceMi is how far the station looks for a relay.
func (lp *livePipelines) bedFenceMi() float64 {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if lp.bedRadiusMi <= 0 {
		return config.DefaultBedRadiusMi
	}
	return lp.bedRadiusMi
}

// stepBedRelay moves the operator's selection and tunes what they land on.
//
// IT WRAPS, because a selector that stopped at the ends would leave the operator
// pressing a key that does nothing and wondering which of the two reasons it
// was. With one relay in reach it is a no-op that stays on the one relay, which
// is the honest answer to a fence that reaches one thing — and what the reach
// advice warns about before they get here (D-77).
// IT RETURNS A COMMAND (D-79). Tuning reaches the player and the row reaches the
// program, and the swap's own stop already taught this lesson once: anything
// that talks back to the program must not run on the Update loop.
//
// THE SELECTION MOVES INLINE, THOUGH. Which relay is chosen has to be TRUE by
// the time the next frame draws, and only the two things that TALK are deferred.
func (lp *livePipelines) stepBedRelay(by int) tea.Cmd {
	relays := lp.bedRelays()
	if len(relays) == 0 {
		return nil
	}
	lp.mu.Lock()
	next := ((lp.bedPick+by)%len(relays) + len(relays)) % len(relays)
	lp.bedPick = next
	chosen := relays[next]
	// AND THE CHOICE IS REMEMBERED, not merely published (F-98, D-90). It was
	// published and nothing else, so the SETTLE — which publishes the same row
	// from the Director's bed, on every tick — overwrote it about a second later
	// and the operator's selection reverted to "(no relay tuned)" no matter what
	// they picked. One fact needs ONE owner, and this is it: the only thing that
	// tunes the station's bed is the only thing that says what it is tuned to.
	lp.bedRelay = relayLine(chosen)
	line := lp.bedRelay
	p := lp.p
	lp.mu.Unlock()

	return func() tea.Msg {
		// THE DECK TUNES IT THROUGH THE ONE FUNCTION THAT ALREADY DOES THIS. The
		// relay-fault window answers with a call sign the same way (MVS-D-76),
		// so a second tuning path here would be a second place for the duck to
		// be lifted.
		if lp.deck != nil {
			lp.deck.tuneCallsign(chosen.Callsign)
		}
		// AND THE CONSOLE IS TOLD WHAT IT LANDED ON. The Director publishes what
		// the bed is CARRYING; this is what the operator has SELECTED, which is
		// a different fact until they cut to it.
		if p != nil {
			p.Send(tty.BedMsg{Relay: line, Carrying: lp.bedCarrying()})
		}
		return nil
	}
}

// selectedRelay is the relay the operator chose, and "" until they choose one.
//
// IT IS THE ROW'S ONE ANSWER (F-98, D-90). The settle asks this rather than
// carrying its own idea, which is the same fix `noteBedCarrying` already made for
// whether the bed is CARRYING — two publishers of one row, and the frequent one
// did not know what the operator had done.
//
// EMPTY IS A REAL ANSWER, not a missing one: until the operator touches the
// selector, what the bed is on is the monitor's rotation, and the Director's label
// is the only honest thing to say about it.
func (lp *livePipelines) selectedRelay() string {
	if lp == nil {
		return ""
	}
	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.bedRelay
}

// bedCarrying is whether the bed holds the programme, as the console last heard.
func (lp *livePipelines) bedCarrying() bool {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.bedOn
}

// relayLine is how a relay reads on the bed's row, from the reference:
// `KIG78 Coachella CA 162.400 MHz · 41mi from TOWER GPS`.
func relayLine(n stream.Near) string {
	mi := strconv.FormatFloat(n.KM*0.621371, 'f', 0, 64)
	return n.Callsign + " " + n.Site + " " + n.State + " " + n.FreqMHz + " MHz · " + mi + "mi from TOWER GPS"
}

// noteBedCarrying records what the Director says the bed is doing, so the relay
// selector's own message carries the same answer.
//
// RECORDED, NEVER DECIDED. The Director owns whether the bed carries; this is
// the console's two publishers agreeing about it rather than one of them
// guessing and the row flickering between them.
func (lp *livePipelines) noteBedCarrying(carrying bool) {
	lp.mu.Lock()
	lp.bedOn = carrying
	lp.mu.Unlock()
}
