package tty

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// PropagationResult is one Propagation update's snapshot, or why there is
// none (0.19.0 W4.1).
type PropagationResult struct {
	Snapshot ionomaps.Snapshot
	Err      error
}

// propUpdatedMsg is an update's result, to the session of the mode that
// asked; propTickMsg is the refresh tick, to the same.
type (
	propUpdatedMsg struct {
		gen     uint64
		res     PropagationResult
		correct bool // the no-readings correction it was asked with (D-105)
	}
	propTickMsg struct{ gen, seq uint64 }
)

// propSession is one open Propagation mode's own state (BUILD red team
// C-3): made as the mode opens and dropped whole when it is left, so nothing
// of a left session is carried into the next.
type propSession struct {
	ctx       context.Context
	cancel    context.CancelFunc // ends the session's update
	busy      bool               // an update asked and not yet answered
	tickEvery time.Duration      // the refresh chain's wait; zero with none set (D-120)
	applied   int                // snapshots applied, GIRO's following NOAA's at once not counted: past one, a refresh (FR-3.6)
}

// askPropagation asks for the mode's update (A-30):
// only in the Propagation mode, with the map the window shown and its
// acknowledgement closed (FR-4.2, D-81), and one at a time. The seam is
// called in the command, never on the UI goroutine (FR-4.8).
func (d Dashboard) askPropagation() (Dashboard, tea.Cmd) {
	seam := d.cfg.PropagationUpdate
	ready := seam != nil && !d.mapPane.propSess.busy && d.modal == modalMap && d.mapMode() == modePropagation
	if !ready {
		return d, nil
	}
	if d.mapPane.propSess.ctx == nil {
		d.mapPane.propSess.ctx, d.mapPane.propSess.cancel = context.WithCancel(context.Background())
	}
	d.mapPane.propSess.busy = true
	d.mapPane.gen++
	ctx, gen, opts := d.mapPane.propSess.ctx, d.mapPane.propGen, d.propOptions()
	return d, func() tea.Msg {
		return propUpdatedMsg{gen: gen, res: <-seam(ctx, opts), correct: opts.CorrectNoReadings}
	}
}

// propOptions are the update's options as the Settings stand: the reference
// circuit's noise (D-187) and the no-readings correction (D-105).
func (d Dashboard) propOptions() ionomaps.UpdateOptions {
	return ionomaps.UpdateOptions{Noise: d.mapPropNoise, CorrectNoReadings: d.mapPropCorrect, NoStations: d.mapPropNoGIRO}
}

// armPropOptions asks for the mode's update when the snapshot shown was made
// with other options than the Settings' - another noise (D-187), the
// no-readings correction (D-105) or the stations (D-214) asked otherwise: go-ionomaps answers it
// from what it holds, asking nothing. A changed Setting is so shown as soon
// as the map is.
func (d Dashboard) armPropOptions() (Dashboard, tea.Cmd) {
	snap := d.mapPane.propSnap
	if snap == nil || (snap.Noise == d.mapPropNoise && d.mapPane.propSnapCorrect == d.mapPropCorrect && snap.Inputs.NoStations == d.mapPropNoGIRO) {
		return d, nil
	}
	return d.askPropagation()
}

// armPropRefresh sets the session's next tick by the refresh Setting when it
// changed while the mode is open (D-120): hourly from every 10 minutes, or
// none on demand; a chain set before it asks nothing.
func (d Dashboard) armPropRefresh() (Dashboard, tea.Cmd) {
	open := d.mapPane.propSess.ctx != nil && !d.mapPane.propSess.busy && d.mapPane.propSnap != nil
	if !open || d.mapPane.propSess.tickEvery == d.mapPropRefresh.every() {
		return d, nil
	}
	return d.nextPropTick()
}

// stopPropagation ends the mode's session: its update cancelled, a late
// result or tick dropped. The last snapshot is kept to show at once when the
// mode opens again.
func (d Dashboard) stopPropagation() Dashboard {
	if d.mapPane.propSess.cancel != nil {
		d.mapPane.propSess.cancel()
	}
	d.mapPane.propSess = propSession{}
	d.mapPane.propGen++
	return d.markFound(nil) // a found place is not kept past the mode (D-27)
}

// applyPropUpdated keeps an update's result and sets the next refresh tick.
func (d Dashboard) applyPropUpdated(v propUpdatedMsg) (tea.Model, tea.Cmd) {
	if v.gen != d.mapPane.propGen {
		return d, nil // a left session's
	}
	d.mapPane.propSess.busy = false
	d.mapPane.gen++
	if propHasField(v.res.Snapshot) {
		snap := v.res.Snapshot
		same := d.mapPane.propSnap != nil && sameSnapshot(*d.mapPane.propSnap, snap) && d.mapPane.propSnapCorrect == v.correct
		d.mapPane.propSnap, d.mapPane.propSnapCorrect = &snap, v.correct
		if snap.Offset.Measured {
			d.problem(propOffsetWords(snap.Offset)) // the diagnostics' record of each live offset (FR-5.4)
		}
		if snap.Inputs.Rejected > 0 {
			d.problem(fmt.Sprintf("Propagation: GIRO's last burst sent %d rows the range checks refused", snap.Inputs.Rejected)) // counted, never on the field (FR-5.2)
		}
		if !same { // a refresh answered too soon is the snapshot held: its answers and hours ahead stand (P-4)
			d.mapPane.propBuilt++                                       // a snapshot rebuilt at the same moment - the correction, the stations - is another (L-F3)
			d.mapPane.propAheadSnaps, d.mapPane.propAheadAsked = nil, 0 // the hours ahead are the old snapshot's
		}
		if snap.Reason != ionomaps.StationsNext {
			d.mapPane.propSess.applied++ // GIRO's stations, asked at once, complete the first answer (FR-3.6)
		}
		d.mapPane.propUpdatedAt = d.now()
	}
	d.mapPane.propErr = v.res.Err
	if v.res.Err == nil && v.res.Snapshot.Reason == ionomaps.StationsNext {
		nd, cmd := d.setProp().renderMap().askPropagation() // NOAA's field is shown; GIRO's stations at once (go-ionomaps D-63)
		if cmd != nil {
			return nd, cmd
		}
		return nd.nextPropTick()
	}
	return d.setProp().renderMap().nextPropTick()
}

// sameSnapshot reports whether a snapshot is the one held: of the same moment
// and made with the same options, as a refresh answered too soon is.
func sameSnapshot(held, got ionomaps.Snapshot) bool {
	return held.Computed.Equal(got.Computed) && held.Noise == got.Noise && held.Inputs.NoStations == got.Inputs.NoStations
}

// nextPropTick sets the session's next refresh tick by the Setting (FR-4.10,
// D-120), the one tick that asks: any set before it asks nothing, so one
// chain of ticks runs however many updates were asked. On demand it sets
// none: only the refresh-now key asks.
func (d Dashboard) nextPropTick() (Dashboard, tea.Cmd) {
	d.mapPane.propTickSeq++
	every := d.mapPropRefresh.every()
	d.mapPane.propSess.tickEvery = every
	if every == 0 {
		return d, nil
	}
	tick := propTickMsg{gen: d.mapPane.propGen, seq: d.mapPane.propTickSeq}
	return d, tea.Tick(every, func(time.Time) tea.Msg { return tick })
}

// applyPropTick asks again on the refresh tick; with another window over the
// map it waits for the next.
func (d Dashboard) applyPropTick(v propTickMsg) (tea.Model, tea.Cmd) {
	if v.gen != d.mapPane.propGen {
		return d, nil
	}
	if v.seq != d.mapPane.propTickSeq {
		return d, nil // a later tick is set
	}
	nd, cmd := d.askPropagation()
	if cmd != nil {
		return nd, cmd
	}
	return d.nextPropTick()
}

// propStatusWords are the Propagation mode's status: asking, what the last
// snapshot was made from and when, or why there is none - the library's
// words, which say when it asks again (D-124).
func (d Dashboard) propStatusWords() string {
	snap, err := d.mapPane.propSnap, d.mapPane.propErr
	switch {
	case snap != nil:
		age, _, _ := d.propAge(*snap)
		words := snapshotWords(*snap, d.mapPane.propLayer) + " · " + age // FR-5.1, FR-5.2
		if f := d.propFailedWords(); f != "" {
			words = "Propagation · " + f + " · " + strings.TrimPrefix(words, "Propagation · ") // FR-5.2: a failed update first, over the field kept
		}
		return words
	case errors.Is(err, context.DeadlineExceeded):
		return "Propagation · NOAA did not answer in time · " + d.firstKey(actMapRefresh) + " refreshes now" // the words, never the context's error text
	case err != nil && !errors.Is(err, context.Canceled):
		return "Propagation · " + strings.TrimPrefix(err.Error(), "ionomaps: ") + " · " + d.firstKey(actMapRefresh) + " refreshes now" // D-120: the key that asks again
	case d.mapPane.propSess.busy:
		return "Propagation · asking NOAA for the newest grid"
	}
	return propagationStatus
}

// propStaleAfter is how old the oldest input may be before the field is
// said stale (FR-5.2, A-56): an hour, go-ionomaps' own currency for a GloTEC
// grid - past it the field falls to the climatology, named - and for a GIRO
// reading (D-184).
const propStaleAfter = time.Hour

// propOldestInput is when the oldest input in use was valid (FR-5.1, A-30):
// GloTEC's grid where it is the background, and the oldest GIRO reading
// assimilated; false with neither - the climatology alone.
func propOldestInput(s ionomaps.Snapshot) (time.Time, bool) {
	var oldest time.Time
	if s.Background.FoF2 == ionomaps.GloTEC && !s.Inputs.GloTECValid.IsZero() {
		oldest = s.Inputs.GloTECValid
	}
	if r := s.Inputs.OldestReading; s.Inputs.Stations > 0 && !r.IsZero() && (oldest.IsZero() || r.Before(oldest)) {
		oldest = r
	}
	return oldest, !oldest.IsZero()
}

// propAge says the field's age (FR-5.1, FR-5.2): its oldest input's, "data
// 25 min old", "data just in", or past an hour "STALE: data 2 h 5 min old";
// "climatology" with no measured input. The age is for the freshness tone,
// zero for the climatology.
func (d Dashboard) propAge(s ionomaps.Snapshot) (words string, age time.Duration, stale bool) {
	oldest, ok := propOldestInput(s)
	if !ok {
		return "climatology", 0, false
	}
	age = max(d.now().Sub(oldest), 0)
	words = "data just in"
	if age >= time.Minute {
		words = "data " + spanWords(age) + " old"
	}
	if age > propStaleAfter {
		return "STALE: " + words, age, true
	}
	return words, age, false
}

// spanWords is a span to the minute, "25 min" or "2 h 5 min".
func spanWords(span time.Duration) string {
	m := int(span.Minutes())
	if m < 60 {
		return strconv.Itoa(m) + " min"
	}
	if m%60 == 0 {
		return strconv.Itoa(m/60) + " h"
	}
	return strconv.Itoa(m/60) + " h " + strconv.Itoa(m%60) + " min"
}

// propOffsetNote is the offset's state said beside the field (FR-5.4,
// D-105): GloTEC unusually far from the stations now, or the typical offset
// applied while no readings are held; empty otherwise.
func propOffsetNote(o ionomaps.Offset) string {
	switch {
	case o.Apart():
		return "GloTEC unusually far from the stations now"
	case o.Corrected:
		return "corrected by the typical offset"
	}
	return ""
}

// propOffsetWords are the offset as the Status window and the diagnostics
// say it (FR-5.4): the live offset - GloTEC's foF2 against the stations' -
// beside its typical value and spread, and whether they are unusually far
// apart; with no live offset, why, and whether the typical one is applied.
func propOffsetWords(o ionomaps.Offset) string {
	typical := fmt.Sprintf("typical %+.2f MHz, spread %.2f", o.TypicalMHz, o.SpreadMHz)
	if !o.Measured {
		w := "Propagation offset (GloTEC against the stations): no live offset, fewer than five readings; " + typical
		if o.Corrected {
			w += "; the typical offset is applied"
		}
		return w
	}
	w := fmt.Sprintf("Propagation offset (GloTEC against the stations): live %+.2f MHz; %s", o.LiveMHz, typical)
	if o.Apart() {
		w += "; unusually far apart now"
	}
	return w
}

// snapshotWords say the layer drawn, when the snapshot was computed and why
// it is not new if it is not, then what it was made from (FR-5.1): the most
// needed first, as a narrow window cuts the line's end.
func snapshotWords(s ionomaps.Snapshot, layer int) string {
	words := []string{"Propagation", propLayerName(layer), "at " + s.Computed.UTC().Format("15:04 UTC")}
	switch s.Reason {
	case ionomaps.TooSoon:
		words = append(words, "asked too soon: the last field")
	case ionomaps.Offline:
		words = append(words, "NOAA not reached: the last field")
	case ionomaps.StationsNext:
		words = append(words, "asking GIRO's stations")
	case ionomaps.GIROPaused:
		words = append(words, "GIRO paused after a refusal: the readings it gave first")
	case ionomaps.Unfinished:
		words = append(words, "the update stopped before every file came: built from what was held")
	}
	if len(s.Inputs.Unread) > 0 {
		words = append(words, "could not read NOAA's "+strings.Join(s.Inputs.Unread, ", ")+": the last held used") // FR-5.2
	}
	switch {
	case s.Inputs.Stations > 0:
		words = append(words, fmt.Sprintf("%d stations, oldest %s", s.Inputs.Stations, s.Inputs.OldestReading.UTC().Format("15:04 UTC")))
	case s.Inputs.NoStations:
		words = append(words, "no stations by choice: the background alone") // D-214
	case s.Reason != ionomaps.StationsNext:
		words = append(words, "no station readings this hour: the background alone")
	}
	switch s.Background.FoF2 {
	case ionomaps.GloTEC:
		words = append(words, "foF2 from GloTEC at "+s.Inputs.GloTECValid.UTC().Format("15:04 UTC"))
	case ionomaps.Climatology:
		words = append(words, "foF2 from the climatology: no GloTEC grid")
	}
	words = append(words, "M(3000)F2 from the "+s.Background.M3000.String())
	if n := propOffsetNote(s.Offset); n != "" {
		words = append(words, n) // FR-5.4
	}
	return strings.Join(words, " · ")
}
