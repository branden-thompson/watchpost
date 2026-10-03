package app

// bedrelay.go — the relays the station may carry on its bed, and the operator's
// walk through them (D-78).
//
// THE SELECTOR THE REFERENCE DRAWS: `[ ← ] KIG78 Coachella CA 162.400 MHz · 41mi
// from TOWER GPS [ → ]`. What it walks is the STATION's fence (D-77), never the
// listener's watchlist — which is the HUM LEAD's Lone Pine objection, and the
// reason the bed is the station's business at all.
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
	"github.com/branden-thompson/watchpost/platform/units"
)

// bedRelays is what the station can actually carry, nearest first.
//
// REMEMBERED, NOT DERIVED, AND D-117 IS WHY. The list is not a pure function of
// the transmitter, the bed's fence and the embedded table: it is the answer to
// "which of them does a directory actually stream". That is network work; it is
// resolved when the AREA MOVES (`refreshBedRelays`) and read from here.
//
// THE STORED COPY CANNOT DRIFT: it is replaced whole every time the station's
// region changes, which is the only thing that can invalidate it.
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
// THE EMBEDDED TABLE IS NOT THE ANSWER. It lists every NOAA transmitter in the
// country; being IN it says a tower exists, not that anything relays it to the
// internet. A selector walking that table would let the operator choose a
// callsign nothing streams — dead air, chosen from a list that promised
// otherwise.
//
// `Resolver.order` IS THE CROSS-CHECK: it drops any transmitter the directories
// carry no mount for, prefers the ones COVERING the station's SAME area, and
// orders the rest by distance. Asked at the transmitter rather than at the
// listener, it answers exactly the HUM LEAD's question.
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
		if units.MilesOf(st.KM) <= radiusMi {
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
	// THE SELECTION FOLLOWS THE RELAY, NOT ITS POSITION: matched by its
	// callsign whenever the list moves - the kept one at launch included
	// (D-214) - and cleared while it is not near the station.
	lp.bedPick, lp.bedRelay = 0, ""
	for i, s := range st { // the relays near the station (P10-02)
		if lp.bedCall != "" && s.Callsign == lp.bedCall {
			lp.bedPick, lp.bedRelay = i, relayLine(s)
		}
	}
	p, line, carrying := lp.p, lp.bedRelay, lp.bedOn
	lp.mu.Unlock()
	if p != nil {
		// TWO FACTS, TWO MESSAGES, AND THIS IS THE ONLY WRITER OF THE SECOND
		// (D-125). On `BedMsg` the count would have two other publishers that do
		// not know it and leave it at zero — retracting this answer and disabling a
		// bed that is carrying.
		p.Send(tty.BedMsg{Relay: line, Carrying: carrying})
		p.Send(tty.BedRelaysMsg{Count: len(st)})
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

// stepBedRelay moves the operator's selection, and keeps it (D-214). It plays
// nothing: the operator hears the relay when they play it or go on air (D-215).
//
// IT WRAPS, because a selector that stopped at the ends would leave the operator
// pressing a key that does nothing and wondering which of the two reasons it
// was. With one relay in reach it is a no-op that stays on the one relay, which
// is the honest answer to a fence that reaches one thing — and what the reach
// advice warns about before they get here (D-77).
// IT RETURNS A COMMAND (D-79). The row reaches the program and the choice
// reaches the file, and anything that talks back to the program must not run
// on the Update loop.
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
	// AND THE CHOICE IS REMEMBERED, not merely published (F-98, D-90). Published
	// alone, it would be overwritten by the SETTLE — which publishes the same row
	// from the Director's bed, on every tick — about a second later, and the
	// operator's selection would revert to "(no relay tuned)". The selector is
	// the one owner of what is selected; playing it is `toggleBedRelay`'s and
	// MasterControl's (D-215).
	lp.bedRelay, lp.bedCall = relayLine(chosen), chosen.Callsign
	line, call := lp.bedRelay, lp.bedCall
	p := lp.p
	lp.mu.Unlock()

	return func() tea.Msg {
		_ = savePreference(func(cfg *config.Config) { cfg.Broadcaster.BedRelay = call }) // kept (D-214); a failed write keeps the session's choice
		// AND THE CONSOLE IS TOLD WHAT IT LANDED ON. The Director publishes what
		// the bed is CARRYING; this is what the operator has SELECTED, which is
		// a different fact until they cut to it.
		if p != nil {
			p.Send(tty.BedMsg{Relay: line, Carrying: lp.bedCarrying()})
		}
		return nil
	}
}

// keepBedRelay is the launch's kept bed relay (D-214): selected once the
// relays near the station are known, while it is among them.
func (lp *livePipelines) keepBedRelay(cfg config.Config) {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	lp.bedCall = cfg.Broadcaster.BedRelay
}

// selectedStation is the relay the operator chose, as the relays near the
// station resolve it, and the rest of them to fall through to.
func (lp *livePipelines) selectedStation() (stream.Station, []stream.Station, bool) {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if lp.bedRelay == "" || lp.bedPick >= len(lp.bedStations) {
		return stream.Station{}, nil, false
	}
	return lp.bedStations[lp.bedPick], lp.bedStations, true
}

// toggleBedRelay plays the selected relay, or stops it when it is the one
// playing - the console's play key (D-215, D-216). The console refuses the key
// while the programme holds the air; here a card being read is refused too,
// since starting the relay would stop it on the one player.
func (lp *livePipelines) toggleBedRelay() tea.Cmd {
	chosen, relays, ok := lp.selectedStation()
	if !ok || lp.deck == nil {
		return nil
	}
	return func() tea.Msg {
		switch {
		case lp.deck.playing(chosen.Callsign):
			lp.deck.stopStation()
		case !lp.deck.reading():
			// THE STATION'S OWN RESOLUTION TUNES IT (D-117), never `tuneCallsign`:
			// `chosen` carries its own mounts, with the station's reach behind it
			// to fall through to.
			lp.deck.tuneResolved(chosen, relays)
		}
		return nil
	}
}

// playBed starts the selected relay - going on air, or cutting the programme
// to the bed (D-215) - unless it already plays or a card is being read.
func (lp *livePipelines) playBed() {
	chosen, relays, ok := lp.selectedStation()
	if !ok || lp.deck == nil || lp.deck.playing(chosen.Callsign) || lp.deck.reading() {
		return
	}
	lp.deck.tuneResolved(chosen, relays)
}

// stopBed stops a bed relay that is playing - standby is dead air (D-215).
func (lp *livePipelines) stopBed() {
	if lp.deck != nil && lp.deck.live() {
		lp.deck.stopStation()
	}
}

// selectedRelay is the relay the operator chose, and "" until they choose one.
//
// IT IS THE ROW'S ONE ANSWER (F-98, D-90). The settle asks this rather than
// carrying its own idea, as it asks `noteBedCarrying` whether the bed is
// CARRYING — two publishers of one row, and the frequent one cannot know what
// the operator has done.
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
	mi := strconv.FormatFloat(units.MilesOf(n.KM), 'f', 0, 64)
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
