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
	"context"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
)

// bedRelays is what the station can actually carry, nearest first.
//
// REMEMBERED, NOT DERIVED, AND D-117 IS WHY. This comment used to read "DERIVED
// ON EVERY ASK, like the pool: a pure function of the transmitter, the bed's
// fence and the embedded table" — true when the list came from a static CSV of
// every NOAA tower in the country, and false the moment the list became the
// answer to "which of them does a directory actually stream". That is network
// work; it is resolved when the AREA MOVES (`refreshBedRelays`) and read from
// here.
//
// THE DRIFT THE OLD COMMENT FEARED IS REAL AND IS HANDLED WHERE IT ARISES: the
// stored copy is replaced whole every time the station's region changes, which
// is the only thing that can invalidate it.
func (lp *livePipelines) bedRelays() []stream.Station {
	if lp == nil {
		return nil
	}
	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.bedStations
}

// refreshBedRelays asks which relays actually STREAM near the station, and
// remembers the answer (D-117).
//
// HUM LEAD, 2026-09-13: "When we derive the station lineup from the service
// radius - we should probably cross-check that against our weatherradio.us and
// wxradio feeds to see if any of that list is a valid relay in the feed. If none
// exist in that area - we should probably tell the broadcaster there is no valid
// relays for their area and disable the BED option so the Operator cannot choose
// something that will broadcast dead air."
//
// THE EMBEDDED TABLE IS NOT THE ANSWER, and that is the whole defect. It lists
// every NOAA transmitter in the country; being IN it says a tower exists, not
// that anything relays it to the internet. The selector walked that table, so the
// operator could choose a callsign nothing streams — and `tuneCallsign` then
// searched the LISTENER's last tune list, failed to find it, and returned in
// silence. Dead air, chosen from a list that promised otherwise.
//
// `Resolver.order` IS THE CROSS-CHECK, already written: it drops any transmitter
// the directories carry no mount for, prefers the ones COVERING the station's
// SAME area, and orders the rest by distance. Asked at the transmitter rather
// than at the listener, it answers exactly the HUM LEAD's question.
//
// IT IS ASYNC AND CACHED, because it is network work and `bedRelays` is asked on
// every frame. The area moving is what re-asks it.
func (lp *livePipelines) refreshBedRelays(ctx context.Context) {
	if lp == nil || lp.deck == nil {
		return
	}
	s := lp.currentStation()
	if s.transmitter.Lat == 0 && s.transmitter.Lon == 0 {
		lp.setBedStations(nil) // no epicentre, no region, no relays to choose between
		return
	}
	lp.setBedStations(withinBedFence(lp.deck.resolveAt(ctx, s.transmitter), lp.bedFenceMi()))
}

// withinBedFence keeps the relays inside the station's reach.
//
// THE BED'S OWN FENCE, NOT THE RESOLVER'S CAP. The resolver ranks the whole
// country by distance and stops at a candidate COUNT; the bed asks what is
// within the station's REACH, which is a different question and the one D-77
// ruled — "Lone Pine's 'Fresno' relay is completely inappropriate for being the
// relay".
//
// A FUNCTION OF ITS OWN so the rule stays testable without a network: what
// `refreshBedRelays` adds around it is the resolve, and that is the part a unit
// test has no business doing.
func withinBedFence(stations []stream.Station, radiusMi float64) []stream.Station {
	kept := make([]stream.Station, 0, len(stations))
	for _, st := range stations { // bounded by the candidate cap (P10-02)
		if st.KM*0.621371 <= radiusMi {
			kept = append(kept, st)
		}
	}
	return kept
}

// setBedStations records the resolved list and tells the console what it can do.
//
// THE CONSOLE IS TOLD, NOT LEFT TO INFER. Whether the bed may be cut to at all
// is a fact about the station's REGION, and the console holds no region — so it
// arrives the way the relay and the carrying state already do.
func (lp *livePipelines) setBedStations(st []stream.Station) {
	lp.mu.Lock()
	lp.bedStations = st
	if lp.bedPick >= len(st) {
		lp.bedPick = 0 // the list moved under the selection
	}
	// AND A SELECTION THAT NO LONGER EXISTS IS CLEARED. A remembered relay from a
	// region the station has left would sit on the row looking tuned.
	if len(st) == 0 {
		lp.bedRelay = ""
	}
	p, line, carrying := lp.p, lp.bedRelay, lp.bedOn
	lp.mu.Unlock()
	if p != nil {
		p.Send(tty.BedMsg{Relay: line, Carrying: carrying, Relays: len(st)})
	}
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
		// THE STATION'S OWN RESOLUTION TUNES IT (D-117). This called
		// `tuneCallsign`, which searches the mount list the LISTENER's last tune
		// left behind and returns in SILENCE when the callsign is not in it — so
		// the bed did nothing at all unless Observer happened to have tuned that
		// same relay. The operator pressed the key, the row said it was tuned, and
		// the station carried dead air.
		//
		// `chosen` IS A RESOLVED STATION and carries its own mounts, so the engine
		// is pointed at them directly, with the rest of the station's reach behind
		// it to fall through to — which is what `tuneRef` does for the listener,
		// through the same function.
		if lp.deck != nil {
			lp.deck.tuneResolved(chosen, relays)
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
func relayLine(n stream.Station) string {
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
