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

	"github.com/branden-thompson/watchpost/modes/tty/internal/frame"
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
// only in the Propagation mode, with the map open - in front or under
// another window, as the bands chart opened over it (D-339) - and its
// acknowledgement closed (FR-4.2, D-81), and one at a time. The seam is
// called in the command, never on the UI goroutine (FR-4.8).
func (mm mapModel) askPropagation(e mapEnv) (mapModel, tea.Cmd) {
	seam := mm.mapHooks.PropagationUpdate
	ready := seam != nil && !mm.mapPane.propSess.busy && e.Shown && e.Window != mapModalPropAck && mm.mapMode(e) == modePropagation
	if !ready {
		return mm, nil
	}
	if mm.mapPane.propSess.ctx == nil {
		mm.mapPane.propSess.ctx, mm.mapPane.propSess.cancel = context.WithCancel(context.Background())
	}
	mm.mapPane.propSess.busy = true
	mm.mapPane.gen++
	ctx, gen, opts := mm.mapPane.propSess.ctx, mm.mapPane.propGen, mm.propOptions()
	return mm, func() tea.Msg {
		return propUpdatedMsg{gen: gen, res: <-seam(ctx, opts), correct: opts.CorrectNoReadings}
	}
}

// propOptions are the update's options as the Settings stand: the reference
// circuit's noise (D-187) and the no-readings correction (D-105).
func (mm mapModel) propOptions() ionomaps.UpdateOptions {
	return ionomaps.UpdateOptions{Noise: mm.mapPropNoise, CorrectNoReadings: mm.mapPropCorrect, NoStations: mm.mapPropNoGIRO}
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
func (mm mapModel) madeFromHeld(s ionomaps.Snapshot) propMade {
	return propMade{computed: s.Computed, built: mm.mapPane.propBuilt, opts: madeWith(s, mm.mapPane.propSnapCorrect)}
}

// armPropOptions asks for the mode's update when the snapshot shown was made
// with other options than the Settings' - another noise (D-187), the
// no-readings correction (D-105) or the stations (D-214) asked otherwise: go-ionomaps answers it
// from what it holds, asking nothing. A changed Setting is so shown as soon
// as the map is.
func (mm mapModel) armPropOptions(e mapEnv) (mapModel, tea.Cmd) {
	snap := mm.mapPane.propSnap
	if snap == nil || madeWith(*snap, mm.mapPane.propSnapCorrect) == mm.propOptions() {
		return mm, nil
	}
	return mm.askPropagation(e)
}

// armPropRefresh sets the session's next tick by the refresh Setting when it
// changed while the mode is open (D-120): hourly from every 10 minutes, or
// none on demand; a chain set before it asks nothing.
func (mm mapModel) armPropRefresh(e mapEnv) (mapModel, tea.Cmd) {
	open := mm.mapPane.propSess.ctx != nil && !mm.mapPane.propSess.busy && mm.mapPane.propSnap != nil
	if !open || mm.mapPane.propSess.tickEvery == mm.mapPropRefresh.every() {
		return mm, nil
	}
	return mm.nextPropTick(e)
}

// stopPropagation ends the mode's session: its update and any find in
// flight cancelled, a late result, answer or tick dropped. The last snapshot
// is kept to show at once when the mode opens again.
func (mm mapModel) stopPropagation(e mapEnv) mapModel {
	if mm.mapPane.propSess.cancel != nil {
		mm.mapPane.propSess.cancel()
	}
	mm = mm.dropFind() // a found place is not kept past the mode, nor one found after it (D-27)
	mm.mapPane.propSess = propSession{}
	mm.mapPane.propGen++
	return mm.markFound(nil, e) // a found place is not kept past the mode (D-27)
}

// applyPropUpdated keeps an update's result and sets the next refresh tick.
func (mm mapModel) applyPropUpdated(v propUpdatedMsg, e mapEnv) (mapModel, tea.Cmd) {
	if v.gen != mm.mapPane.propGen {
		return mm, nil // a left session's
	}
	mm.mapPane.propSess.busy = false
	mm.mapPane.gen++
	if propHasField(v.res.Snapshot) {
		snap := v.res.Snapshot
		same := mm.mapPane.propSnap != nil && sameSnapshot(*mm.mapPane.propSnap, mm.mapPane.propSnapCorrect, snap, v.correct)
		newlyDenied := snap.Inputs.GIRO == ionomaps.GIRODenied && (mm.mapPane.propSnap == nil || mm.mapPane.propSnap.Inputs.GIRO != ionomaps.GIRODenied)
		var was ionomaps.Inputs // the snapshot held's: a count is noted when it changes, never again for each snapshot of the hour (R3L-7)
		if mm.mapPane.propSnap != nil {
			was = mm.mapPane.propSnap.Inputs
		}
		mm.mapPane.propSnap, mm.mapPane.propSnapCorrect = &snap, v.correct
		if newlyDenied {
			mm.problem("Propagation: GIRO refused this host (a 401 or 403): it is not asked again for a day", e) // D-247, D-272: named once, as it begins
		}
		if snap.Offset.Measured {
			mm.problem(propOffsetWords(snap.Offset), e) // the diagnostics' record of each live offset (FR-5.4)
		}
		if snap.Inputs.Rejected > 0 && snap.Inputs.Rejected != was.Rejected {
			mm.problem(fmt.Sprintf("Propagation: GIRO's last burst sent %d rows or replies that were refused: past the range checks, unreadable, or from a station placed far from where it is", snap.Inputs.Rejected), e) // counted, never on the field (FR-5.2)
		}
		if snap.Inputs.Implausible > 0 && snap.Inputs.Implausible != was.Implausible {
			mm.problem(fmt.Sprintf("Propagation: %d station readings were refused as implausible, their foF2 too far from the background to be believed (D-248, D-279)", snap.Inputs.Implausible), e)
		}
		if snap.Inputs.ImplausibleM > 0 && snap.Inputs.ImplausibleM != was.ImplausibleM {
			mm.problem(fmt.Sprintf("Propagation: %d station readings' M(3000)F2 was refused as implausible; their foF2 was used (D-248)", snap.Inputs.ImplausibleM), e)
		}
		if !same { // a refresh answered too soon is the snapshot held: its answers and hours ahead stand (P-4)
			mm.mapPane.propBuilt++                                        // a snapshot rebuilt at the same moment - the correction, the stations - is another (L-F3)
			mm.mapPane.propAheadSnaps, mm.mapPane.propAheadAsked = nil, 0 // the hours ahead are the old snapshot's
		}
		if snap.Reason != ionomaps.StationsNext {
			mm.mapPane.propSess.applied++ // GIRO's stations, asked at once, complete the first answer (FR-3.6)
		}
		mm.mapPane.propUpdatedAt = e.Now()
	}
	mm.mapPane.propErr = v.res.Err
	if v.res.Err == nil && v.res.Snapshot.Reason == ionomaps.StationsNext {
		nd, cmd := mm.setProp(e).renderMap(e).askPropagation(e) // NOAA's field is shown; GIRO's stations at once (go-ionomaps D-63)
		if cmd != nil {
			return nd, cmd
		}
		return nd.nextPropTick(e)
	}
	return mm.setProp(e).renderMap(e).nextPropTick(e)
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
func (mm mapModel) nextPropTick(e mapEnv) (mapModel, tea.Cmd) {
	mm.mapPane.propTickSeq++
	every := mm.mapPropRefresh.every()
	mm.mapPane.propSess.tickEvery = every
	if every == 0 {
		return mm, nil
	}
	tick := propTickMsg{gen: mm.mapPane.propGen, seq: mm.mapPane.propTickSeq}
	return mm, tickAfter(e.Ticks, every, tick)
}

// applyPropTick asks again on the refresh tick, a window over the map or not
// (D-339): a chart left open refreshes with the map under it.
func (mm mapModel) applyPropTick(v propTickMsg, e mapEnv) (mapModel, tea.Cmd) {
	if v.gen != mm.mapPane.propGen {
		return mm, nil
	}
	if v.seq != mm.mapPane.propTickSeq {
		return mm, nil // a later tick is set
	}
	nd, cmd := mm.askPropagation(e)
	if cmd != nil {
		return nd, cmd
	}
	return mm.nextPropTick(e)
}

// propStatusWords are the Propagation mode's status: asking, what the last
// snapshot was made from and when, or why there is none - the library's
// words, which say when it asks again (D-124).
func (mm mapModel) propStatusWords(e mapEnv) string {
	snap, err := mm.mapPane.propSnap, mm.mapPane.propErr
	switch {
	case snap != nil:
		age, _, _ := mm.propAge(*snap, e)
		words := snapshotWords(*snap, mm.mapPane.propLayer) + " · " + age // FR-5.1, FR-5.2
		if f := mm.propFailedWords(); f != "" {
			words = "Propagation · " + f + " · " + strings.TrimPrefix(words, "Propagation · ") // FR-5.2: a failed update first, over the field kept
		}
		return words
	case errors.Is(err, context.DeadlineExceeded):
		return "Propagation · NOAA did not answer in time · " + mm.firstKey(actMapRefresh) + " refreshes now" // the words, never the context's error text
	case err != nil && !errors.Is(err, context.Canceled):
		return "Propagation · " + strings.TrimPrefix(err.Error(), "ionomaps: ") + " · " + mm.firstKey(actMapRefresh) + " refreshes now" // D-120: the key that asks again
	case mm.mapPane.propSess.busy:
		return "Propagation · asking NOAA for the newest grid"
	}
	return propagationStatus
}

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
// 25 min old", "data just in", or past the refresh's bound (staleAfter)
// "STALE: data 2 h 5 min old";
// "climatology" with no measured input. The age is for the freshness tone,
// zero for the climatology.
func (mm mapModel) propAge(s ionomaps.Snapshot, e mapEnv) (words string, age time.Duration, stale bool) {
	oldest, ok := propOldestInput(s)
	if !ok {
		return "climatology", 0, false
	}
	age = max(e.Now().Sub(oldest), 0)
	words = "data just in"
	if age >= time.Minute {
		words = "data " + spanWords(age) + " old"
	}
	if age > mm.mapPropRefresh.staleAfter() {
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
	if days := int(age.Hours() / 24); days >= 1 {
		return strconv.Itoa(days) + " day" + frame.Plural(days)
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
