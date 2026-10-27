package tty

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

// propMade is the snapshot an answer is worked out from, as it was kept: its
// moment, which snapshot kept - one rebuilt at the same moment is another
// (L-F3) - and the options it was made with. Every answer's key carries it
// and the options are compared in this one place (RC-5); a new option is
// added to propOptions and to madeWith, which reads it back from the
// snapshot, and TestEveryUpdateOptionIsMadeWith fails until it is (R2C-2).
type propMade struct {
	computed time.Time
	built    uint64
	opts     ionomaps.UpdateOptions
}

// madeWith is the options a snapshot was made with: its noise and stations
// as it says them, and the no-readings correction it was asked with (D-105),
// which it does not carry.
func madeWith(s ionomaps.Snapshot, correct bool) ionomaps.UpdateOptions {
	return ionomaps.UpdateOptions{Noise: s.Noise, CorrectNoReadings: correct, NoStations: s.Inputs.NoStations}
}

// madeFromHeld is the key of answers worked out from s - the snapshot held,
// or an hour ahead made from it: s's moment and noise, with the held
// snapshot's build and its no-readings correction, which s does not carry.
func (d Dashboard) madeFromHeld(s ionomaps.Snapshot) propMade {
	return propMade{computed: s.Computed, built: d.mapPane.propBuilt, opts: madeWith(s, d.mapPane.propSnapCorrect)}
}

// armPropOptions asks for the mode's update when the snapshot shown was made
// with other options than the Settings' - another noise (D-187), the
// no-readings correction (D-105) or the stations (D-214) asked otherwise: go-ionomaps answers it
// from what it holds, asking nothing. A changed Setting is so shown as soon
// as the map is.
func (d Dashboard) armPropOptions() (Dashboard, tea.Cmd) {
	snap := d.mapPane.propSnap
	if snap == nil || madeWith(*snap, d.mapPane.propSnapCorrect) == d.propOptions() {
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

// stopPropagation ends the mode's session: its update and any find in
// flight cancelled, a late result, answer or tick dropped. The last snapshot
// is kept to show at once when the mode opens again.
func (d Dashboard) stopPropagation() Dashboard {
	if d.mapPane.propSess.cancel != nil {
		d.mapPane.propSess.cancel()
	}
	d = d.dropFind() // a found place is not kept past the mode, nor one found after it (D-27)
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
		same := d.mapPane.propSnap != nil && sameSnapshot(*d.mapPane.propSnap, d.mapPane.propSnapCorrect, snap, v.correct)
		newlyDenied := snap.Inputs.GIRO == ionomaps.GIRODenied && (d.mapPane.propSnap == nil || d.mapPane.propSnap.Inputs.GIRO != ionomaps.GIRODenied)
		var was ionomaps.Inputs // the snapshot held's: a count is noted when it changes, never again for each snapshot of the hour (R3L-7)
		if d.mapPane.propSnap != nil {
			was = d.mapPane.propSnap.Inputs
		}
		d.mapPane.propSnap, d.mapPane.propSnapCorrect = &snap, v.correct
		if newlyDenied {
			d.problem("Propagation: GIRO refused this host (a 401 or 403): it is not asked again for a day") // D-247, D-272: named once, as it begins
		}
		if snap.Offset.Measured {
			d.problem(propOffsetWords(snap.Offset)) // the diagnostics' record of each live offset (FR-5.4)
		}
		if snap.Inputs.Rejected > 0 && snap.Inputs.Rejected != was.Rejected {
			d.problem(fmt.Sprintf("Propagation: GIRO's last burst sent %d rows or replies that were refused: past the range checks, unreadable, or from a station placed far from where it is", snap.Inputs.Rejected)) // counted, never on the field (FR-5.2)
		}
		if snap.Inputs.Implausible > 0 && snap.Inputs.Implausible != was.Implausible {
			d.problem(fmt.Sprintf("Propagation: %d station readings were refused as implausible, their foF2 too far from the background to be believed (D-248, D-279)", snap.Inputs.Implausible))
		}
		if snap.Inputs.ImplausibleM > 0 && snap.Inputs.ImplausibleM != was.ImplausibleM {
			d.problem(fmt.Sprintf("Propagation: %d station readings' M(3000)F2 was refused as implausible; their foF2 was used (D-248)", snap.Inputs.ImplausibleM))
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
func sameSnapshot(held ionomaps.Snapshot, heldCorrect bool, got ionomaps.Snapshot, gotCorrect bool) bool {
	return held.Computed.Equal(got.Computed) && madeWith(held, heldCorrect) == madeWith(got, gotCorrect)
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

// giroWords say how GIRO's last burst ended, carried in every snapshot
// until the next (D-247): empty when it answered.
func giroWords(g ionomaps.GIROOutcome) string {
	switch g {
	case ionomaps.GIRODenied:
		return "GIRO refused this host: not asked again for a day"
	case ionomaps.GIROUnavailable:
		return "GIRO unavailable this hour"
	}
	return ""
}

// stationsThinAfter is how old the newest station reading may be before the
// map is said to rest on stale data (D-319).
const stationsThinAfter = 3 * time.Hour

// thinStationsWords say, as kc2g does with a banner, that the map is a best
// approximation: fewer than half the stations holding a reading gave one in
// the last 3 hours, or the newest is older than that; empty otherwise, and
// when no reading is held, which the words below say as the background
// alone (D-316, D-319).
func thinStationsWords(in ionomaps.Inputs, at time.Time) string {
	if in.NoStations || in.StationsHeld == 0 {
		return ""
	}
	if age := at.Sub(in.NewestReading); age > stationsThinAfter {
		return "BEST APPROXIMATION: the newest station reading is " + spanWords(age) + " old; rely on this map warily"
	}
	if in.StationsRecent*2 < in.StationsHeld {
		return fmt.Sprintf("BEST APPROXIMATION: %d of %d stations reported in the last 3 hours; rely on this map warily", in.StationsRecent, in.StationsHeld)
	}
	return ""
}

// daysWords is an age said in days past a day, else as spanWords.
func daysWords(age time.Duration) string {
	switch days := int(age.Hours() / 24); {
	case days == 1:
		return "1 day"
	case days > 1:
		return strconv.Itoa(days) + " days"
	}
	return spanWords(age)
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
	if w := thinStationsWords(s.Inputs, s.Computed); w != "" {
		words = append(words, w) // D-319: said first, as kc2g's banner is
	}
	switch s.Reason {
	case ionomaps.TooSoon:
		words = append(words, "asked too soon: the last field")
	case ionomaps.Offline:
		words = append(words, "nothing came back from NOAA: built from what is held") // D-239: never more than happened
	case ionomaps.StationsNext:
		words = append(words, "asking GIRO's stations")
	case ionomaps.GIROPaused:
		w := giroWords(s.Inputs.GIRO)
		if w == "" {
			w = "GIRO paused after a refusal"
		}
		words = append(words, w+": the readings it gave first")
	case ionomaps.Unfinished:
		words = append(words, "the update stopped before every file came: built from what was held")
	}
	if w := giroWords(s.Inputs.GIRO); w != "" && s.Reason != ionomaps.GIROPaused {
		words = append(words, w) // the last burst's end, said all hour (D-247)
	}
	if len(s.Inputs.Unread) > 0 { // a Partial snapshot's, or one GIRO's state said first (D-239)
		words = append(words, "NOAA's "+strings.Join(s.Inputs.Unread, ", ")+" not read this update") // FR-5.2: what stands in is said below, the background by its source
	}
	if slices.Contains(s.Inputs.Unread, ionomaps.UnreadSolar) && s.Inputs.F107Age > 0 {
		words = append(words, "the solar flux held from "+daysWords(s.Inputs.F107Age)+" ago") // D-249: the kept mean, its age said
	}
	switch {
	case s.Inputs.Stations == 1:
		words = append(words, "1 station, at "+s.Inputs.OldestReading.UTC().Format("15:04 UTC")) // R9L-3
	case s.Inputs.Stations > 1:
		words = append(words, fmt.Sprintf("%d stations, oldest %s", s.Inputs.Stations, s.Inputs.OldestReading.UTC().Format("15:04 UTC")))
	case s.Inputs.NoStations:
		words = append(words, "no stations by choice: the background alone") // D-214
	case s.Reason != ionomaps.StationsNext:
		words = append(words, "no station readings this hour: the background alone")
	}
	switch n := s.Inputs.StationsFailed; {
	case n == 1:
		words = append(words, "1 station gave no data") // D-316: a station's miss, the burst going on
	case n > 1:
		words = append(words, fmt.Sprintf("%d stations gave no data", n))
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
