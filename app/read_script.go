package app

// read_script.go — the Reader (MVS-D-77, T3.8).
//
// ONE READER, TWO CALLERS. The live takeover reads through it today; the
// Director's Speak executor reads through it at T3.10. Two implementations of
// MVS-D-72's pacing would be two places for the ruling to drift, and the pacing
// is the thing a listener actually hears — the HUM LEAD found the old pacing by
// ear, on a real alert, not by any gate.
//
// WHAT IT OWNS: the shape of a read. The tone, the pauses between parts, and the
// overlap that renders the next part while this one sounds, so no gap contains a
// render — the ~1 s that made a takeover "feel broken".
//
// WHAT IT DOES NOT OWN: the words (the Composer), which alerts are in the burst
// (the Producer), or when it is read (the Director). It is handed a script and
// performs it.

import (
	"fmt"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// gapAfter is MVS-D-72's structure, in ONE place: how long the Reader waits
// after a part before the next begins.
//
// AND IT IS ONE PLACE NOW. That claim was false when it was written — the shape
// was here and all four numbers were in the PRODUCER's file, so someone told to
// change the pause between alerts edited this switch and found nothing to
// change (red team 2026-09-05, Junior-Dev 10). The constants moved to the
// bottom of this file on 2026-09-06 rather than the claim being softened: the
// Reader owns pacing (S-7), so the numbers belong with the rule that reads
// them.
//
//	<tone>  2 s  <header>  1 s  <alert-1>  1 s  …  <alert-n>  2 s  <tail>
//
// The 2 s after the tone is absent because it is not a wait: the tone preset
// carries its own two-second tail inside the buffer, so holding the tone's own
// duration holds the pause with it.
func gapAfter(kind lineup.PartKind, lastLine bool) time.Duration {
	switch kind {
	case lineup.PartHead:
		return burstHeadGap // header → first alert
	case lineup.PartLine:
		if lastLine {
			return burstTailGap // last alert → closing tail
		}
		return breakingGap // between alerts (MVS-D-66)
	}
	return 0 // nothing follows the tail
}

// readHooks are what the caller does around a part it is about: put the band's
// callout up before the words, and record it read afterwards. A structural part
// (a header, a tail) is about the burst rather than any one alert and carries no
// Ref, so neither hook fires for it.
type readHooks struct {
	cue  func(ref string)
	mark func(ref string)
}

// reader carries the state one read needs: the look-ahead clip rendered during
// the part before it, and where the alert lines begin and end.
//
// A STRUCT RATHER THAN SIX ARGUMENTS. The look-ahead crosses iterations, which
// is what made a single function do too much — P10 measured readScript at 20
// against a bound of 15. Naming the state lets each step be its own step.
type reader struct {
	s         *speaker
	h         readHooks
	next      clip
	haveNext  bool
	firstLine int
	lastLine  int

	// toned and spoke are FR-9.2's two facts: a tone was sounded, and how many
	// parts reached the air after it. A tone is a promise of words, and this is
	// the only place that can tell whether the promise was kept.
	toned bool
	spoke int
}

// readScript performs one card. It returns false when the sequence ended — the
// context was cancelled, or the job left the air — and the caller stops.
func readScript(s *speaker, sc lineup.Script, h readHooks) bool {
	r := &reader{s: s, h: h, firstLine: firstLineIndex(sc), lastLine: lastLineIndex(sc)}
	if !r.openTone(sc) {
		return false
	}
	for i, p := range sc.Parts { // bounded by the script (P10-02)
		if !r.readPart(sc, i, p) {
			return false
		}
	}
	r.reportSilentTone()
	return true
}

// reportSilentTone says so when a tone was sounded and nothing followed it
// (FR-9.2).
//
// A TONE IS A PROMISE OF WORDS. The listener hears the attention tone, leans
// in, and gets nothing — the failure reported three times in one burst at UAT
// 2026-09-06 and impossible to diagnose from the outside, because a read that
// ends early is Routed and raises nothing (I-2).
//
// THE THREE LEGITIMATE TONES-WITH-NO-WORDS NEVER REACH HERE, and the check for
// them would be a branch with no failing input (D-2). An operator pressing esc,
// a takeover pre-empting the read and the pump stopping all CANCEL, and every
// path to this line has already checked the air: openTone returns false when
// its hold finds the context gone, and readPart checks before every callout. To
// arrive here is to have passed an air check, so a guard on ctx.Err() would
// read as extra safety and be extra surface — the shape relayfault.go's own
// empty guard had.
//
// A planted mutation proved it: removing that guard broke nothing, because
// nothing could construct the state it excluded.
//
// ONLY WHEN NOTHING WAS SPOKEN. A read that got a line out and then lost
// the rest is a different failure with a different shape, and calling it this
// one would train the operator to read past both.
func (r *reader) reportSilentTone() {
	if !r.toned || r.spoke > 0 {
		return
	}
	if r.s.d == nil || r.s.d.v == nil {
		return
	}
	r.s.d.v.fault("an alert tone sounded and no words followed it")
}

// openTone sounds the attention tone and renders the first part behind it, so
// the words begin the moment the tone ends rather than a render later.
func (r *reader) openTone(sc lineup.Script) bool {
	toneStart := time.Now()
	toneDur := time.Duration(0)
	if c, ok := cast.ClassByKey(sc.Tone); ok && sc.Tone != "" {
		toneDur = r.s.attention(c)
		r.toned = toneDur > 0 // sounded, not merely asked for: a muted or voiceless job tones nothing
	}
	// prepare only RENDERS; nothing is spoken early, and the cue still precedes
	// the words.
	if len(sc.Parts) > 0 {
		r.next, r.haveNext = timeRender(sc.Parts[0].Kind.String(), func() (clip, bool) { return r.s.prepare(sc.Parts[0].Text) })
	}
	// THE ONE PLACE A TONE CAN BE SOUNDED AND NOTHING SAID, so it is the one
	// place worth a line in the timeline (F-43).
	//
	// A takeover's band callout comes from INSIDE the read, per line (cueFor),
	// so a read that gives up here produces a tone with no callout and no words
	// — heard three times in one burst at UAT 2026-09-06 and impossible to
	// diagnose from the outside, because a cut-short read is Routed and
	// therefore raises nothing (I-2). The listener hears a promise and gets
	// silence; the log now says where the promise was broken.
	if ok := holdRest(r.s, toneDur, toneStart); ok {
		return true
	}
	if radioDebugOn() {
		radioDebugLog(fmt.Sprintf("read:gaveup:after-tone tone=%s dur=%s parts=%d ctx=%v",
			sc.Tone, toneDur, len(sc.Parts), r.s.ctx.Err()))
	}
	return false
}

// readPart performs one part: the retry that earns its place, the callout, the
// words, the render of what follows, and the pause the ruling gives it.
func (r *reader) readPart(sc lineup.Script, i int, p lineup.Part) bool {
	// ONE RETRY, FOR THE FIRST LINE ONLY, and it earns its place on exactly one
	// path: a live render that failed on its OWN — no voice for the role, or the
	// synthesiser erroring — where it is the only thing between a line and
	// silence.
	//
	// It cannot help the common case. `prepare` also refuses when the job is
	// inaudible or the director silent (a muted takeover), and when the text is
	// empty; re-asking with the same text satisfies neither. The look-ahead
	// renders have no retry — a failure there falls to the fixed hold, which is
	// what keeps this from becoming a render loop.
	if !r.haveNext && i == r.firstLine {
		r.next, r.haveNext = timeRender("retry", func() (clip, bool) { return r.s.prepare(p.Text) })
	}
	// THE AIR IS CHECKED BEFORE EVERY CALLOUT, not only after a wait: a cue is a
	// promise that words are coming, and a sequence that ended during the render
	// before it would otherwise put a headline on the band for a read that never
	// happens.
	if !r.s.awaitAir(nil) {
		return false
	}
	if p.Ref != "" && r.h.cue != nil {
		r.h.cue(p.Ref)
	}
	cur, have := r.next, r.haveNext
	hold, start := breakingHold, time.Now()
	if have {
		if d := r.s.deliver(cur); d > 0 {
			hold, r.spoke = d, r.spoke+1
		}
	}
	// d == 0 means muted, or no voice at runtime — keep the fixed hold so the
	// part is still readable rather than blitted past (P4 F10).
	if i+1 < len(sc.Parts) { // render the NEXT while this one sounds
		n := sc.Parts[i+1]
		r.next, r.haveNext = timeRender(n.Kind.String(), func() (clip, bool) { return r.s.prepare(n.Text) })
	}
	if !holdRest(r.s, hold, start) { // whatever is LEFT after that render
		return false
	}
	if p.Ref != "" && r.h.mark != nil {
		r.h.mark(p.Ref)
	}
	// NOTHING BOUNDS THE READ FROM HERE (DR-3). Once a card is admitted it is
	// read in full; the only bound is the Max the Producer applied.
	if g := gapAfter(p.Kind, i == r.lastLine); g > 0 && i+1 < len(sc.Parts) {
		return r.s.hold(g)
	}
	return true
}

// firstLineIndex is the first alert line, which is the only part with a retry.
// -1 when the script has no lines at all.
func firstLineIndex(sc lineup.Script) int {
	for i, p := range sc.Parts { // bounded by the script (P10-02)
		if p.Kind == lineup.PartLine {
			return i
		}
	}
	return -1
}

// lastLineIndex is where the alert lines end, so the Reader knows which gap
// precedes the tail. -1 when the script has no lines at all.
func lastLineIndex(sc lineup.Script) int {
	last := -1
	for i, p := range sc.Parts { // bounded by the script (P10-02)
		if p.Kind == lineup.PartLine {
			last = i
		}
	}
	return last
}

// --- MVS-D-72's pause structure, and the wait that honours it ---
//
// MOVED HERE FROM ticker.go (2026-09-06). The Reader's own header claimed
// gapAfter was "MVS-D-72's structure, in ONE place" while all four numbers —
// breakingHold and the three gaps — lived in the PRODUCER's file, so someone
// told to change the pause between alerts edited the switch and found nothing
// to change (red team, Junior-Dev 10). The rule and its numbers are one place
// now, and the claim is true. A pure move: no value changed.

// breakingHold is the fallback centre-hold per event when there is no audio to
// pace it (muted, or no voice) — the visual still steps (HUM LEAD 2026-08-27).
const breakingHold = 5 * time.Second

// The burst's pause structure, as the HUM LEAD heard it and ruled on it
// (MVS-D-72, UAT 2026-09-03):
//
//	<tone>  2 s  <header>  1 s  <alert-1>  1 s  …  <alert-n>  2 s  <tail>
//
// THESE ARE WHAT THE LISTENER HEARS. A takeover's clip is speech and nothing
// else, so the wait IS the pause.
//
// An earlier version subtracted synth.SegmentGap from each, on the premise that
// "every spoken clip already ends with 400 ms of silence inside its own buffer".
// That is true of the BROADCAST segment stream — synth.Source.write pads between
// segments — and false of this path: a takeover renders through
// speaker.prepare → radioDeck.render → synth.AlertNarration, which is Say plus a
// stereo conversion and pads nothing. Every gap in the burst therefore came out
// 400 ms short of its ruling, measured at 0.6 s where MVS-D-72 says 1 s. The
// premise was never checked against the path that uses it.
//
// The 2 s after the tone is not here because it is not a wait: the tone preset
// carries its own two-second tail inside the buffer, so holding the tone's own
// duration holds the pause with it.
const (
	burstHeadGap = time.Second     // header → first alert
	breakingGap  = time.Second     // between alerts (MVS-D-66, unchanged)
	burstTailGap = 2 * time.Second // last alert → closing tail
)

// holdRest waits what is LEFT of d, given that work has already been running
// since start.
//
// THE POINT OF OVERLAPPING IS LOST WITHOUT IT. A render started during the tone
// and then a full-length hold would simply move the dead air rather than remove
// it — the listener would wait the tone AND the render, which is what this whole
// change exists to stop. A negative remainder means the work outran the sound and
// there is nothing left to wait.
func holdRest(s *speaker, d time.Duration, start time.Time) bool {
	rest := d - time.Since(start)
	if rest <= 0 {
		// NOTHING LEFT TO WAIT IS NOT THE SAME AS NOTHING TO CHECK. s.hold(0)
		// returns true without reaching its own air check, so a sequence that
		// ended while the work overran would carry on to the next line.
		return s.awaitAir(nil)
	}
	return s.hold(rest)
}
