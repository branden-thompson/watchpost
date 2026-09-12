package app

// radio_queue.go — the watchlist queue and the live-relay dwell: [r] Repeat, advancing, the nearest-station pick. Split from radio.go by the quality pass (Q2, pure move).

import (
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// chooseNearest picks the station Nearest Relay mode plays (UAT 97): the
// resolver lists the covering transmitter first when it is relayed, then
// the nearest relayed ones — so the first with a mount is the answer.
//
// EXCEPT WHERE TWO STATIONS ARE EQUALLY THE ANSWER. Coachella KIG78 and
// Coachella / Spanish WNG712 share a mast: same coordinates, same covering
// status, so nothing about the geography prefers either and the resolver's
// order between them is a tie-break, not a finding. Vista, CA got the Spanish
// feed that way (HUM LEAD, UAT 2026-09-04), and the ruling was that the choice
// is the listener's: "some humans will prefer english, some will prefer
// spanish".
//
// So the preference decides the TIE and nothing else. A listener who prefers
// Spanish does not get a Spanish station 200 km away over the English one in
// their county — the nearest relay is still the nearest relay, and this only
// answers the question the distance leaves open.
func chooseNearest(stations []stream.Station, prefer string) (stream.Station, bool) {
	first := -1
	for i, st := range stations { // bounded by the candidate list (P10-02)
		if len(st.Mounts) > 0 {
			first = i
			break
		}
	}
	if first < 0 {
		return stream.Station{}, false
	}
	best := stations[first]
	if prefer == "" || best.Lang() == prefer {
		return best, true
	}
	for _, st := range stations[first+1:] { // bounded by the candidate list (P10-02)
		// Only a station that is EQUALLY near and equally covering can take
		// the place: anything else is a different answer, not a tie.
		if st.KM != best.KM || st.Covering != best.Covering {
			break
		}
		if len(st.Mounts) > 0 && st.Lang() == prefer {
			return st, true
		}
	}
	return best, true
}

// SetRepeat implements tty.Radio (UAT 83/93): One loops the synthesized
// broadcast; Watchlist lets the current cycle end and then advances through
// the queue (a live relay advances after liveDwell); Off plays to the end.
func (d *radioDeck) SetRepeat(mode tty.RepeatMode, watchlist []snapshot.LocationRef) {
	d.mu.Lock()
	d.repeat, d.queue = mode, watchlist
	src, ref := d.source, d.ref
	d.mu.Unlock()
	// THE DIRECTOR IS STILL TOLD; THE LIVE SOURCE IS NOT TOUCHED (D-91).
	//
	// THIS IS THE ONE THE HUM LEAD'S RULING WAS MEASURED ON: `d.source` is
	// whatever is running, and per BD-9 that is the BROADCASTER's card during a
	// main-track read — so Observer's repeat mode, reached from the Settings
	// window the console forwards to, set the card ON THE AIR to loop. It would
	// have read for ever and the line-up would never have advanced.
	//
	// THE SETTING STILL APPLIES, which is the `SetTones` standard: "must not
	// disturb a broadcast in flight". It lands on the monitor's next source.
	if src != nil && d.monitorHasTheAir() {
		src.Loop(mode == tty.RepeatOne)
	}
	// THE ROTATION IS THE DIRECTOR'S NOW (T3.2b). A zero dwell is how "repeat is
	// not Watchlist" reaches it — the translation from the deck's mode enum
	// happens here, once, rather than by a second copy of the enum living in the
	// pure core. It is told on every change, playing or not: the Director holds
	// the rotation, and a setting it never heard is a setting that does not
	// apply.
	dwell := time.Duration(0)
	if mode == tty.RepeatWatchlist {
		dwell = d.watchlistDwell()
	}
	keys := make([]string, 0, len(watchlist))
	for _, r := range watchlist { // bounded by the watchlist (P10-02)
		keys = append(keys, string(snapshot.Key(r)))
	}
	d.tell(lineup.Programme{Watchlist: keys, Dwell: dwell})
	_ = ref
}
