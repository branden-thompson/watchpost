package tty

// locate.go — the debounced "can the station reach this place?" check, and the
// bubbletea half of platform/debounce.
//
// ONE MECHANISM, TWO FIELDS (D-130). The console's `[l]` search box and the
// Line-Up Request window's Location field ask the same question of the same
// hook and draw the same sentence about the answer. A second copy of the
// sequence bookkeeping is exactly the kind of duplicate that drifts into two
// different ideas of what "valid" means.

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/debounce"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// locateField names which input a pause or an answer belongs to, so one
// mechanism can serve every field without their answers crossing.
type locateField uint8

const (
	locateLookup  locateField = iota // the console's [l] search box
	locateRequest                    // the Line-Up Request window's Location field
)

// locatePauseMsg says an input has been quiet for debounce.Pause.
type locatePauseMsg struct {
	field locateField
	seq   int
}

// locateVerdictMsg is the hook's answer about one query.
type locateVerdictMsg struct {
	field  locateField
	seq    int
	query  string
	ref    snapshot.LocationRef
	within bool
	found  bool

	// asked is whether the question reached an answer at all (D-151). False
	// means the check could not be MADE — a timeout, a dropped connection, a
	// cancelled context — which is a different thing from "no such place" and
	// must not be drawn as one.
	asked bool
}

// afterPause is the tea half of the debounce, GENERIC OVER THE MESSAGE so any
// future field can wait on its own.
//
// tea.Tick RATHER THAN A CANCELLABLE TIMER, which is the design platform's Gate
// is built around: the tick always fires, and the sequence it carries is what
// decides whether anybody still wants it.
func afterPause[M tea.Msg](seq int, mk func(int) M) tea.Cmd {
	return tea.Tick(debounce.Pause, func(time.Time) tea.Msg { return mk(seq) })
}

// locateState is one field's knowledge about what has been typed into it.
//
// THE QUERY IS KEPT BESIDE THE ANSWER because the window draws them together,
// and an answer shown against different text is the same lie as a stale frame.
type locateState struct {
	gate   debounce.Gate
	query  string // the text the answer below is ABOUT
	ref    *snapshot.LocationRef
	within bool
	found  bool

	// asked is whether the question reached an answer at all (D-151). False
	// after settling means the check could not be MADE — a timeout, a dropped
	// connection, a cancelled context — which is a different thing from "no
	// such place" and must not be drawn as one.
	asked bool

	// submitted is an ENTER pressed before the answer arrived (D-141).
	//
	// THE PRESS IS HELD, NOT DISCARDED AND NOT OBEYED. Discarding it makes the
	// key inert while the field thinks — the dead control D-129 exists to
	// prevent. Obeying it was worse and is what shipped: the not-yet-known
	// branch fell through to the UNSCOPED resolver, so the console's scope was
	// escapable by being quick, which is the very defect D-129 was filed for.
	//
	// SO THE PRESS BRINGS THE ANSWER FORWARD and waits for it: the scoped hook
	// is asked at once rather than at the end of the pause, and the verdict
	// decides. A refusal is still a refusal; an acceptance opens.
	submitted bool
}

// edit records a keystroke and returns the state plus the pause to wait on.
//
// WHAT IT KNEW IS DISCARDED HERE, NOT WHEN THE NEXT ANSWER LANDS. Keeping the
// old verdict through the pause would draw "Outside the service radius" under
// text the operator has already corrected.
func (st locateState) edit(f locateField, query string) (locateState, tea.Cmd) {
	st.gate = st.gate.Edit()
	st.query, st.ref, st.within, st.found, st.asked = query, nil, false, false, false
	st.submitted = false // a new keystroke supersedes a press waiting on the old text
	seq := st.gate.Seq()
	return st, afterPause(seq, func(s int) locatePauseMsg { return locatePauseMsg{field: f, seq: s} })
}

// settled reports whether the field has an answer about what it currently
// holds. Until it does, the window says nothing rather than guessing.
func (st locateState) settled() bool { return st.gate.Settled() }

// apply takes an answer if it is still the one being waited for.
func (st locateState) apply(v locateVerdictMsg) locateState {
	gate, took := st.gate.Settle(v.seq)
	if !took {
		return st
	}
	ref := v.ref
	st.gate, st.found, st.within, st.asked = gate, v.found, v.within, v.asked
	if v.found {
		st.ref = &ref
	} else {
		st.ref = nil
	}
	return st
}

// reachable is the one test every caller makes: a real place the station can
// broadcast about.
func (st locateState) reachable() bool { return st.found && st.within && st.ref != nil }

// locateCmd asks the hook, OFF THE KEY PATH. The hook may reach the network —
// it is the only thing that knows the small places — so it must never be called
// from a handler or a render.
func (d Dashboard) locateCmd(f locateField, seq int, query string) tea.Cmd {
	look := d.cfg.LocateInRadius
	if look == nil || query == "" {
		return nil
	}
	return func() tea.Msg {
		ref, within, found, asked := look(query)
		return locateVerdictMsg{field: f, seq: seq, query: query, ref: ref,
			within: within, found: found, asked: asked}
	}
}

// locateNote is what a field says about its answer, in the shared wording.
func (st locateState) locateNote() (string, string) {
	if !st.settled() {
		return "", "" // asked and not yet answered: the window does not know
	}
	// THE QUESTION COULD NOT BE PUT (D-151). Saying "not found" here tells the
	// operator a real place does not exist, on the evidence of a timeout — and
	// the chip that would let them retry is disabled by the same answer.
	if !st.asked {
		return "Could not check this location.", "The lookup did not answer; press enter to try again"
	}
	if !st.found {
		return poolNote(st.query, nil, false)
	}
	return poolNote(st.query, st.ref, !st.within)
}

// couldNotAsk reports the fourth state: settled, and the check never happened.
func (st locateState) couldNotAsk() bool { return st.settled() && !st.asked }

// submitAnswer is what [enter] means for a location field in the state it is in.
type submitAnswer int

const (
	// submitAsk — the field has no answer yet. Ask, and HONOUR THE PRESS when
	// the verdict lands (D-141): the check is 300 ms of pause plus a geocoder
	// round trip, and a key that went inert while the field was thinking is the
	// dead control this whole rule exists to prevent.
	submitAsk submitAnswer = iota
	// submitRetry — the question could not be PUT (D-151). A timeout answers
	// nothing about the place, so enter asks again rather than going dead.
	submitRetry
	// submitRefuse — a definite no. The chip is drawn unavailable and the key
	// agrees with it, so the window never contradicts its own sentence.
	submitRefuse
	// submitGo — a real place the station can broadcast about, already known.
	// Re-asking would be a second round trip to re-learn it, and a second
	// authority that can disagree with the first.
	submitGo
)

// onSubmit is what [enter] means in this state.
//
// ONE OWNER OF THE FOUR-WAY ANSWER, AT THE SECOND CALLER. The `[l]` window spelt
// these four cases out as an ordered run of `if`s and the Line-Up Request window
// spelt out two of them, which is how the third state went missing: on "could
// not ask" the request window's `valid()` returned false and enter did NOTHING,
// while `locateNote` — shared, and therefore right — printed "The lookup did not
// answer; press enter to try again". A window instructing an action it refuses,
// which is D-151's own defect reached from a third side.
//
// THE ORDER IS THE RULE. "Not settled" outranks everything because a field still
// thinking has no answer to refuse on; "could not ask" outranks "not reachable"
// because a timeout is not a verdict about the place. Reading them as a switch
// rather than a run of `if`s is what makes a missing arm a compile-visible gap
// instead of a silent fall-through.
func (st locateState) onSubmit() submitAnswer {
	switch {
	case !st.settled():
		return submitAsk
	case !st.asked:
		return submitRetry
	case !st.reachable():
		return submitRefuse
	default:
		return submitGo
	}
}

// handleLocatePause is the pause expiring: ask, but only if this is still the
// pause that follows the LAST keystroke.
func (d Dashboard) handleLocatePause(v locatePauseMsg) (tea.Model, tea.Cmd) {
	switch v.field {
	case locateLookup:
		if !d.addLocate.gate.Admits(v.seq) {
			return d, nil // typed over: a later pause is already on its way
		}
		return d, d.locateCmd(v.field, v.seq, d.addLocate.query)
	case locateRequest:
		if !d.request.locate.gate.Admits(v.seq) {
			return d, nil
		}
		return d, d.locateCmd(v.field, v.seq, d.request.locate.query)
	}
	return d, nil
}

// handleLocateVerdict files an answer against the field that asked for it.
func (d Dashboard) handleLocateVerdict(v locateVerdictMsg) (tea.Model, tea.Cmd) {
	switch v.field {
	case locateLookup:
		held := d.addLocate.submitted
		d.addLocate = d.addLocate.apply(v)
		// AND A PRESS THAT WAS WAITING ON THIS ANSWER IS HONOURED (D-141), so
		// the operator never has to press enter twice. A refusal simply leaves
		// the window open with its reason showing.
		if held && d.addLocate.reachable() {
			d.addLocate.submitted = false
			ref := *d.addLocate.ref
			return d, func() tea.Msg { return resolvedMsg{mode: d.addMode, ref: ref} }
		}
	case locateRequest:
		held := d.request.locate.submitted
		d.request.locate = d.request.locate.apply(v)
		// THE SAME HONOURING AS THE LOOKUP BOX (D-151). A press that was waiting
		// on this answer schedules; anything else leaves the window open with
		// its reason showing.
		if held && d.request.locate.reachable() {
			d.request.locate.submitted = false
			return d.requestSchedule()
		}
	}
	return d, nil
}

// keyLabel and keyState are this state's stand-ins in a comparable memo key.
//
// NOT THE POINTER, AND NOT THE STRUCT. A key holding the pointer would differ
// on every rebuild of the same answer; the label is what the frame actually
// shows. `keyState` folds "asked and unanswered", "no such place" and "outside
// the radius" into one value, because those are exactly the three frames this
// field can produce.
func (st locateState) keyLabel() string {
	if st.ref == nil {
		return ""
	}
	return st.ref.Label
}

// keyState is the field's drawable condition, as one comparable value.
func (st locateState) keyState() uint8 {
	switch {
	case !st.settled():
		return 0 // thinking: the window says nothing
	case !st.asked:
		return 4 // the question could not be put (D-151) — a fourth frame
	case !st.found:
		return 1 // no such place
	case !st.within:
		return 2 // a real place the station cannot reach
	}
	return 3 // reachable
}
