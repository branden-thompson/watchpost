package lineup

import (
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Event is something that happened. The clock is one of them (Tick), which is
// what makes determinism structural rather than a seam bolted on (DR-20): there
// is no other way for time to enter the Director.
//
// The marker keeps the set closed: nothing outside this file can be stepped, so
// a mistyped event is a compile error rather than a silent no-op on the hazard
// path. An event is data, never a request.
type Event interface{ event() }

// isEvent carries the marker for every event, ONCE. Embedding it rather than
// writing the method out per type keeps the set closed at the same strength
// while leaving one function where there would otherwise be five, each of them
// empty.
type isEvent struct{}

func (isEvent) event() {}

// Arrived is a set of proposals offered to the burst — Observer has fetched and
// normalised them, and the Director now pre-screens and arranges them (DR-5).
type Arrived struct {
	isEvent
	Arrivals []Arrival

	// Fence is WHERE THE LISTENER IS and how far they care, at the moment these
	// arrivals were noticed. It governs both entry and order (DR-12, DR-13), so
	// it cannot be pre-applied by the producer — Plan owns it.
	//
	// IT RIDES ON THE EVENT rather than sitting in Settings because it is not a
	// standing preference: the origin is the watchlist's first location and the
	// radius is a live setting, both of which can change between one burst and
	// the next. A fence held on the Director would be whatever it was when the
	// Director was built, which for a listener who has since moved the map is
	// the wrong fence applied silently. The zero value is All, unfenced.
	Fence Fence
}

// Tick is the clock, and nothing else. It carries no decision.
type Tick struct {
	isEvent
	Now time.Time
}

// Built is a BuildCard effect coming home with the words it composed (DR-7).
type Built struct {
	isEvent
	ID     string
	Script Script
}

// Finished is a Speak effect coming home: the card was read in full.
type Finished struct {
	isEvent
	ID string
}

// Failed is a card that was not delivered. Producers own their own retries and
// the Director re-routes by default (DR-21); this is what reaches it when the
// card itself did not happen.
type Failed struct {
	isEvent
	ID, Reason string
	// Routed says the non-delivery was DELIBERATE and heals itself — the
	// listener muted the station, the read was cut short, the producer's record
	// had gone. The card did not happen; nothing is wrong with the station.
	//
	// WITHOUT IT THE GRADE COLLAPSES (red team 2026-09-05, I-2). stopped() asks
	// only whether the schedule is empty, and in 0.14.0 it always is once a rail
	// card leaves: MVS-D-77 puts one card per burst on the rail and nothing
	// queues MainTrack. So every deliberate decline reached escalation() with an
	// empty schedule and raised the RELAY FAULT window — a modal telling the
	// listener the relay is dead because they had muted the tones, or pressed
	// esc. fault.go's own header names that hazard: "raising a modal for one
	// would train a listener to dismiss the window that matters — a noise
	// regression, which on a safety surface is a safety regression."
	//
	// THE DEFAULT IS FALSE, so a fault nobody classified is still surfaced. A
	// contained executor panic is the case that leaves it false today.
	Routed bool
}

// Effect is work the Director DESCRIBES and never performs. A pump dispatches
// each one and feeds its result back as an event.
//
// THE SET IS CLOSED (PL-6). It is the architecture's real interface, so adding
// to it is a deliberate change to this file and to what Step can return — not an
// incidental new call site. That is what keeps the Director a coordinator rather
// than something that quietly grows the ability to do work itself.
type Effect interface{ effect() }

// isEffect carries the marker for every effect, ONCE — see isEvent.
type isEffect struct{}

func (isEffect) effect() {}

// BuildCard composes a card's script. THE EXPENSIVE ONE: a cold build is 1.03 s
// of network against a ~0.7 s output buffer, and it is not latency-parallelisable
// (perf-protocol §6), which is the whole reason standby exists.
//
// IT CARRIES THE CARD'S SLOT (BD-8). An executor is chosen by what kind of read
// the card is — a location report is eleven requests and a composer, an alert
// is one line from the producer's record — and the effect is the description of
// that work. Looking the slot up from the published lineup instead would race
// the dispatch: the publish for the same step runs concurrently with this.
type BuildCard struct {
	isEffect
	ID      string
	Slot    Slot
	Subject string
	// Refs are the producer's records this card reads, in read order — the
	// card's own Refs, carried ON THE EFFECT for the reason the slot is (BD-8):
	// looking them up from the published lineup would race the dispatch, since
	// the publish for the same step runs concurrently with the build.
	Refs []string
	// Divert is how many the burst left unread, for the notice that says so
	// (DR-14). It rides here for the reason Refs does (BD-8).
	Divert int
}

// Speak reads a card's words aloud. It carries the slot for the reason
// BuildCard does: which path reads the words — the narrator, or the broadcast
// engine — is decided by what the card is.
type Speak struct {
	isEffect
	ID     string
	Slot   Slot
	Script Script
}

// CueTicker tells the band to show the callout for the card taking the air. Fire
// and trust: the voice never blocks on it (DR-18).
type CueTicker struct {
	isEffect
	ID, Headline string
	// Slot is what kind of read this is, carried for the reason BuildCard's is
	// (BD-8): a card that cues ITSELF as it reads — a takeover, which shows a
	// different callout per alert — must be told apart from one the band shows
	// a single callout for, and the executor cannot ask the lineup without
	// racing the publish for the same step.
	Slot Slot
}

// ReleaseTicker gives the band its rotation back. PAIRED WITH THE CUE (DR-24) —
// every exit from the air emits one.
type ReleaseTicker struct {
	isEffect
	ID string
}

// Duck and Restore give way over the live bed and take it back. NOT EMITTED BY
// ANYTHING IN PRODUCTION — see app/executors.go's Duck case for why that is
// currently harmless and why they are kept (I-5). Originally:
// the bed is a selectable resource rather than a queue, and the Director gains
// it with the main-track absorb (T3.2).
type Duck struct{ isEffect }

// Restore is Duck's other half; see Duck.
type Restore struct{ isEffect }

// Tune cuts the bed over to a relay. EMITTED, at bed.go's dwell and rejoin —
// the "not emitted yet" this carried was written before T1.6 and stayed through
// the release (R-3). Originally: it arrives with the
// running state (T1.6) and the absorb (T3.2).
type Tune struct {
	isEffect
	Ref string
}

// Publish hands the settled lineup to its readers — the synth layer, the fetch
// layer, the ticker, the deck — so each can decide what to pre-load or sync.
//
// IT CARRIES THE LINEUP RATHER THAN NAMING IT. The pump dispatches
// asynchronously, so a reader that fetched the Director's current lineup when
// the effect ran would get whatever it had become by then, not what was
// published.
//
// IT CARRIES THE POWER TOO (0.16.0 P2), and for the same reason it carries
// the lineup by value: a reader told the schedule and the station's state
// SEPARATELY can hold a torn pair — a new lineup beside a stale power — and
// the console's whole job is to show what is actually going to air. One
// message, one consistent moment.
type Publish struct {
	isEffect
	Lineup Lineup
	Power  Power
}

// Describe is what an effect IS, in one line.
//
// ONE OWNER FOR THE FORMAT, not a String method on each of eight types. DR-23
// asks for a timeline that reads as one timeline, and a format defined in eight
// places is a format that drifts in eight places — which is exactly the failure
// the category registry exists to prevent (F-21). Adding an effect without a
// line here returns empty, and the test that walks the closed set catches it.
func Describe(e Effect) string {
	if err := invariant.Check(e != nil, "an effect describes something"); err != nil {
		return ""
	}
	switch v := e.(type) {
	case BuildCard:
		return named("build", v.ID)
	case Speak:
		return named("speak", v.ID)
	case CueTicker:
		return named("cue", v.ID)
	case ReleaseTicker:
		return named("release", v.ID)
	case Duck:
		return "duck()"
	case Restore:
		return "restore()"
	case Tune:
		return named("tune", v.Ref)
	case Escalate:
		return named("escalate", v.ID)
	case Publish:
		return fmt.Sprintf("publish(rail=[%s] main=[%s])",
			strings.Join(idsOf(v.Lineup.Cards(AlertRail)), " "),
			strings.Join(idsOf(v.Lineup.Cards(MainTrack)), " "))
	}
	return ""
}

// CardOf is the card an effect acts on, when it acts on one.
//
// The pump needs it for two things and they are the same question: which
// effects must keep the order Step gave them (those about ONE card — the cue
// before the words, DR-18), and which card to fail when an executor panics
// (DR-22). Publishing, ducking and tuning name no card; concurrency between
// them is free, and a fault in one fails nothing.
func CardOf(e Effect) (string, bool) {
	if err := invariant.Check(e != nil, "an effect is something"); err != nil {
		return "", false
	}
	switch v := e.(type) {
	case BuildCard:
		return v.ID, v.ID != ""
	case Speak:
		return v.ID, v.ID != ""
	case CueTicker:
		return v.ID, v.ID != ""
	case ReleaseTicker:
		return v.ID, v.ID != ""
	}
	return "", false
}

// Resource is something an effect USES while it runs, and that only one effect
// may use at a time. Two effects that share one must keep the order Step gave
// them; two that share none may run at once.
//
// IT LIVES HERE BECAUSE IT IS A PROPERTY OF THE VOCABULARY, not of any pump.
// "The same card" is not the whole rule: the cue must reach the band before the
// words are spoken (DR-18) — one card — and the release of the card LEAVING the
// air must reach it before the cue of the card TAKING it (DR-24), which is two
// different cards sharing one band.
type Resource string

// TheBand is the marquee: it shows one callout at a time, whichever card that
// callout is for.
const TheBand Resource = "band"

// TheBed is the live broadcast the alerts speak over. Ducking it, reading over
// it and restoring it are one another's ordering constraints: a duck that
// landed after the read had started would be the duck-lift bug in a new
// costume (RD-2), and a restore that overtook the read would lift the bed back
// over a correspondent still speaking.
const TheBed Resource = "bed"

// Holds is what an effect uses while it runs. An effect that holds nothing —
// publishing, ducking, tuning — is free to run alongside anything.
func Holds(e Effect) []Resource {
	if err := invariant.Check(e != nil, "an effect is something"); err != nil {
		return nil
	}
	var out []Resource
	if id, ofCard := CardOf(e); ofCard {
		out = append(out, Resource("card:"+id))
	}
	switch e.(type) {
	case CueTicker, ReleaseTicker:
		out = append(out, TheBand)
	case Duck, Restore, Speak:
		out = append(out, TheBed)
	}
	return out
}

// named is one effect and the thing it acts on.
//
// AN EFFECT WITH NO IDENTITY MAKES AN UNREADABLE LINE. "build()" is a line
// nobody can act on, so it says so rather than looking like a rendering bug.
func named(word, id string) string {
	if id == "" {
		return word + "(?)"
	}
	return word + "(" + id + ")"
}

// idsOf is the cards' identities, in order.
func idsOf(cards []Card) []string {
	out := make([]string, 0, len(cards))
	for _, c := range cards { // bounded by the track (P10-02)
		out = append(out, c.ID)
	}
	if err := invariant.Check(len(out) == len(cards), "every card names itself in the published line"); err != nil {
		return nil
	}
	return out
}

// Director owns the Lineup and decides what happens next. It performs nothing.
//
// EXACTLY ONE WRITER (DR-1), by construction rather than by discipline: the
// Lineup is unexported, Step is a pure function returning a NEW Director, and
// only the pump calls it.
type Director struct {
	lineup   Lineup
	settings Settings
	now      time.Time
	power    Power
	bed      bed // what the broadcast rides on, and when it took it (T3.2b)
}

// New is a Director with the listener's settings and a clock already set.
//
// The clock is seeded rather than left zero because planning reads it: a zero
// clock would make every disaster older than the freshness window and demote the
// lot of them, silently and only in the ordering.
func New(s Settings, now time.Time) Director {
	if err := invariant.Check(!now.IsZero(), "a director is built with its clock already set"); err != nil {
		return Director{}
	}
	if err := invariant.Check(s.Max >= 0, "a director is built with a Max that is not negative"); err != nil {
		return Director{}
	}
	return Director{settings: s, now: now}
}

// Lineup is the schedule as it stands — a copy of the value, which readers may
// hold as long as they like.
func (d Director) Lineup() Lineup { return d.lineup }

// Now is the clock as the last Tick left it.
func (d Director) Now() time.Time {
	if err := invariant.Check(!d.now.IsZero(), "the clock was set before it was read"); err != nil {
		return time.Time{}
	}
	return d.now
}

// Step is the whole Director: one event in, the next Director and the work to be
// done out. It is PURE — no I/O, no clock, no goroutines — so a test states an
// arrangement and reads back what would be spoken, with nothing to synchronise
// with (NFR-D-1).
//
// The dispatch stays small and the work lives in one handler per event, from the
// first commit rather than after P10-04 complains: a step function's natural
// shape is one enormous switch, and that is the shape that stops being readable
// exactly when the schedule gets interesting.
func (d Director) Step(ev Event) (Director, []Effect) {
	if err := invariant.Check(!d.now.IsZero(), "the director's clock was set before it was stepped"); err != nil {
		return d, nil
	}
	if err := invariant.Check(ev != nil, "an event is something that happened, never nothing"); err != nil {
		return d, nil
	}
	switch e := ev.(type) {
	case Arrived:
		return d.onArrived(e)
	case Tick:
		return d.onTick(e)
	case Built:
		return d.onBuilt(e)
	case Finished:
		return d.onFinished(e)
	case Failed:
		return d.onFailed(e)
	case Powered:
		return d.onPowered(e)
	case NeedsRead, Offered:
		return d.stepProducer(ev)
	case Tuned, Programme, Ended, CutOver:
		return d.stepBed(ev)
	case Moved, Dropped, Restored:
		return d.stepOperator(ev)
	}
	// An event nothing handles changes nothing. The set is closed, so this is
	// unreachable for anything built here — and it is the safe direction for
	// anything added later without a handler.
	return d, nil
}

// stepProducer routes the Producer's acts — what cards should exist (D-40).
//
// THE GROUP IS A ROLE, not a bucket. The role model's first row is the Card
// Producer, whose whole job is "what cards should exist, and when"; a location
// needing a read and a set of proposals offered are the two ways that reaches
// the Director, and neither of them says a word about what the card SAYS.
//
// It keeps the safe default, as the other two groups do: an event added here
// without a handler falls through to `Step`'s own and changes nothing.
func (d Director) stepProducer(ev Event) (Director, []Effect) {
	switch e := ev.(type) {
	case NeedsRead:
		return d.onNeedsRead(e)
	case Offered:
		return d.onOffered(e)
	}
	return d, nil
}

// stepBed routes what happens to the broadcast the programme rides on.
//
// GROUPED BY CONCERN, not to move a number. `Step` crossed P10-04's branch
// bound when the operator's three acts arrived, and splitting a dispatch into
// two dispatches merely relocates the count — as this project measured once
// already, when seamsPresent came out of newExecutors at the same size. What
// makes this a real split is that the two groups are two subjects, and they are
// the two files these handlers already live in.
//
// EACH GROUP KEEPS THE SAFE DEFAULT. An event added to neither falls through to
// `Step`'s own, which changes nothing — the same direction as before.
func (d Director) stepBed(ev Event) (Director, []Effect) {
	switch e := ev.(type) {
	case Tuned:
		return d.onTuned(e)
	case Programme:
		return d.onProgramme(e)
	case Ended:
		return d.onEnded(e)
	case CutOver:
		return d.onCutOver(e)
	}
	return d, nil
}

// stepOperator routes the human's three acts on a card (FR-3).
func (d Director) stepOperator(ev Event) (Director, []Effect) {
	switch e := ev.(type) {
	case Moved:
		return d.onMoved(e)
	case Dropped:
		return d.onDropped(e)
	case Restored:
		return d.onRestored(e)
	}
	return d, nil
}

// onArrived plans the burst and queues it. Bounds applied at admission and
// nowhere later (DR-3): what is queued here will be read.
func (d Director) onArrived(ev Arrived) (Director, []Effect) {
	// THE LISTENER'S FENCE, THE DIRECTOR'S MAX. The fence arrives with the
	// burst because it is where the listener stood when these alerts did; the
	// Max is a standing preference and stays on the Director.
	s := d.settings
	s.Fence = ev.Fence
	b, err := Plan(ev.Arrivals, s, d.now)
	if err != nil {
		// A malformed proposal set is the producer's bug, and the schedule
		// already on the rail is not its victim. The step is a no-op.
		return d, nil
	}
	held := len(d.lineup.tracks[AlertRail])
	// ONE CARD ON THE RAIL (MVS-D-77). b.Cards is the Producer's ORDERING — the
	// alerts and their read order, which the Composer turns into this card's
	// script — and the schedule holds the takeover, not the alerts. The
	// operator promotes or drops the burst; there is nothing inside it to
	// address separately.
	if b.HasTakeover() {
		if next, err := d.lineup.Queue(AlertRail, b.Takeover); err == nil {
			d.lineup = next // an error is "already held": a repeated burst is not a second read
		}
	}
	// NO ADMITTED CARD IS EVER DROPPED (DR-3). A burst arriving while the rail
	// drains adds to it; it never replaces what is already promised.
	if err := invariant.Check(len(d.lineup.tracks[AlertRail]) >= held, "queueing a burst never shortens the rail"); err != nil {
		return d, nil
	}
	return d.settle()
}

// onTick moves the clock. It decides nothing by itself — but the schedule is
// re-settled, so a card whose turn has come is picked up.
//
// THE CLOCK NEVER GOES BACKWARDS. Ticks can arrive out of order from a pump that
// batches, and a clock that went back would re-age every disaster and quietly
// re-sort the next burst.
func (d Director) onTick(ev Tick) (Director, []Effect) {
	if err := invariant.Check(!ev.Now.IsZero(), "a tick carries a time"); err != nil {
		return d, nil
	}
	was := d.now
	if ev.Now.After(d.now) {
		d.now = ev.Now
	}
	if err := invariant.Check(!d.now.Before(was), "the clock never runs backwards"); err != nil {
		return d, nil
	}
	// THE BED'S DWELL IS A TICK'S BUSINESS TOO (T3.2b). It was a time.AfterFunc
	// inside the radio deck, which made "when does the bed move" observable only
	// by waiting five minutes with a real clock.
	d, moved := d.advanceBed()
	// FR-9.3: a tune the station never made. Asked before settle, so a stall
	// reported this tick is not hidden by whatever the schedule does next.
	d, stalled := d.stalledRotation()
	next, fx := d.settle()
	return next, append(append(moved, stalled...), fx...)
}

// onBuilt puts a card's composed script on it (DR-7). The card is at standby,
// its data was fetched just now, and what it will say is decided from that.
func (d Director) onBuilt(ev Built) (Director, []Effect) {
	if err := invariant.Check(!ev.Script.Empty(), "a build comes home with words on it"); err != nil {
		return d, nil
	}
	card, ok := d.find(ev.ID)
	if !ok {
		return d, nil // discarded while its build was in flight; ordinary
	}
	built, err := card.WithScript(ev.Script, d.now)
	if err != nil {
		return d, nil
	}
	if err := invariant.Check(built.Words() == ev.Script.Text(), "the card carries the words it was built with"); err != nil {
		return d, nil
	}
	next, err := d.lineup.Set(built)
	if err != nil {
		return d, nil
	}
	d.lineup = next
	return d.settle()
}

// onFinished takes a card off the air, read in full.
func (d Director) onFinished(ev Finished) (Director, []Effect) { return d.leave(ev.ID, Done) }

// onFailed takes a card off the schedule: it could not be delivered at all, and
// the Director re-plans around it rather than waiting (DR-21).
// onFailed takes a card off the schedule and grades what that leaves (DR-21).
//
// THE GRADE IS DECIDED AFTER THE SETTLE, not before. A burst whose second alert
// cannot be rendered still has its third to read, and the schedule carries on —
// that is a fault ROUTED AROUND, and it gets today's quiet treatment. The same
// failure on the last card leaves the station silent, which is the one case a
// person has to be told about.
func (d Director) onFailed(ev Failed) (Director, []Effect) {
	d, fx := d.leave(ev.ID, Discarded)
	return d, append(fx, d.escalation(ev)...)
}

// leave is EVERY EXIT A CARD HAS, and the only one — which is what makes DR-24's
// pairing a property rather than a discipline. Today the band's release is sent
// on one path with five early returns above it that send nothing; here there is
// one path, and it releases whatever it cued.
func (d Director) leave(id string, to State) (Director, []Effect) {
	d, fx, left := d.takeOffTheAir(id, to)
	if !left {
		// A completion for a card the schedule no longer holds is ORDINARY —
		// it was discarded or superseded while its work was in flight — and it
		// must not disturb whatever is on the air now, not even by publishing.
		return d, nil
	}
	d, more := d.settle()
	return d, append(fx, more...)
}

// takeOffTheAir is leave WITHOUT the settle that follows it, for the one caller
// that settles afterwards on its own.
//
// A step publishes ONCE, LAST. A caller that both leaves the air and settles
// would otherwise describe two publishes with the first no longer last — and
// the pump runs publishes concurrently, because they hold nothing, so a reader
// could apply the older snapshot after the newer one and show a card that has
// already left the schedule.
func (d Director) takeOffTheAir(id string, to State) (Director, []Effect, bool) {
	if err := invariant.Check(to == Done || to == Discarded, "a card leaves the schedule for a state it cannot come back from"); err != nil {
		return d, nil, false
	}
	card, ok := d.find(id)
	if !ok {
		return d, nil, false
	}
	held := len(d.lineup.tracks[AlertRail]) + len(d.lineup.tracks[MainTrack])
	wasOnAir := card.State == OnAir
	gone, err := card.To(to)
	if err != nil {
		return d, nil, false
	}
	next, err := d.lineup.Set(gone)
	if err != nil {
		return d, nil, false
	}
	if next, err = next.Remove(id); err != nil {
		return d, nil, false
	}
	d.lineup = next
	// A CARD THAT LEFT MUST BE GONE. One still held after its exit is a card the
	// schedule will offer again, which is a second read of something the
	// listener has already heard.
	if err := invariant.Check(len(d.lineup.tracks[AlertRail])+len(d.lineup.tracks[MainTrack]) < held,
		"a card leaving the schedule shortens it"); err != nil {
		return d, nil, false
	}
	var fx []Effect
	if wasOnAir {
		// PAIRED WITH THE CUE, not with the card: releasing a band that was never
		// cued would clear whatever callout it is legitimately showing.
		fx = append(fx, ReleaseTicker{ID: id})
	}
	return d, fx, true
}

// settle is what the schedule does after any change: put a card on the air if
// the air is free, get the next one built, and tell the readers.
//
// ORDER MATTERS AND IS THE DESIGN. Taking the air first means the card that
// follows becomes "next", so its build starts while this one reads — which is
// DR-7 and the 1.03 s finding, structural rather than scheduled. Publishing last
// means a subscriber never sees a state this same step is still changing.
func (d Director) settle() (Director, []Effect) {
	d, air := d.takeTheAir()
	d, prep := d.prepareNext()
	// A card that arrived with its own words reached standby just now, in this
	// same step. If the air is free it takes it here rather than waiting for
	// the next event to come round: the silence between a burst starting and
	// its head being read would otherwise be however long the tick happens to
	// be, which is not a decision this function should be making by accident.
	if len(air) == 0 {
		d, air = d.takeTheAir()
	}
	if _, busy := d.lineup.OnAir(); !busy && len(air) > 0 {
		return d, nil // the air was taken and then lost; publish nothing rather than a lie
	}
	// THE DUCK COMES BEFORE THE CUE, and the effect set's own comment says why:
	// "a duck that landed after the read had started would be the duck-lift bug
	// in a new costume". Decided here, on the settled schedule, so the answer is
	// about what is actually about to happen.
	//
	// AND THE STATE IS COMMITTED ONLY ON THIS PATH. Above, an air that was taken
	// and then lost returns nothing at all; moving the bed's record before that
	// return would leave the Director believing it had ducked while emitting no
	// effect to do it.
	d, give := d.giveOrTakeBack()
	fx := append(give, air...)
	fx = append(fx, prep...)
	fx = append(fx, Publish{Lineup: d.lineup, Power: d.Power()})
	last := fx[len(fx)-1]
	_, published := last.(Publish)
	// THE READERS ARE TOLD LAST. A subscriber reads the lineup to decide what to
	// pre-load, so it must never be handed a state this same step is still
	// changing — which is a rule about the ORDER of a list, and therefore one
	// that can be asserted rather than remembered.
	if err := invariant.Check(published, "the readers are told last, after the schedule has settled"); err != nil {
		return d, nil
	}
	return d, fx
}

// takeTheAir puts the next card on, if the air is free and its words are ready.
//
// THE CUE PRECEDES THE WORDS (DR-18). It is the order of this list, so no call
// site can get it wrong — which is what Phase 0 had to pin by hand (T0.3, m49).
func (d Director) takeTheAir() (Director, []Effect) {
	// TWO PASSES AT MOST, AND THE BOUND IS THE RULE (PD-3, P10-01).
	//
	// The first pass may find the next card stale, drop it and queue the
	// notice; the second puts that notice on the air. There is never a third,
	// because the notice's words were fixed at proposal so it was never built,
	// so it can never itself be stale.
	//
	// This was mutual recursion — takeTheAir calling readInstead calling
	// takeTheAir — and it terminated for exactly that reason. But the reason
	// lived two files away from the call, which is the shape P10-01 exists to
	// refuse: a loop whose termination depends on a fact nothing local states.
	// As a bound it is checkable, and a change that broke it would spin here
	// rather than blow the stack.
	for range 2 {
		next, fx, again := d.airOnce()
		d = next
		if !again {
			return d, fx
		}
	}
	return d, nil
}

// airOnce is one attempt at putting a card on the air. It reports `again` when
// it changed the schedule instead of filling the air — the stale case — so the
// caller may try the card that replaced it.
func (d Director) airOnce() (Director, []Effect, bool) {
	if _, busy := d.lineup.OnAir(); busy {
		return d, nil, false
	}
	next, track, ok := d.lineup.Next()
	if !ok || next.State != Standby || next.Script.Empty() {
		return d, nil, false
	}
	if !d.advances(track) {
		return d, nil, false // the listener stopped the programme (PD-1)
	}
	// PD-3, AND ONLY HERE. Staleness is a question about a card that is about
	// to be READ. A card sitting in standby while the programme is stopped is
	// not stale, it is waiting — which is why this sits below the advances
	// guard and not on a tick.
	if d.now.Sub(next.BuiltAt) > StaleAfter && !next.BuiltAt.IsZero() {
		d, queued := d.readInstead()
		return d, nil, queued
	}
	if err := invariant.Check(!next.Script.Empty(), "a card takes the air with its words already on it"); err != nil {
		return d, nil, false
	}
	onAir, err := next.To(OnAir)
	if err != nil {
		return d, nil, false
	}
	if err := invariant.Check(onAir.State == OnAir, "the card that was cued is the card on the air"); err != nil {
		return d, nil, false
	}
	moved, err := d.lineup.Set(onAir)
	if err != nil {
		return d, nil, false
	}
	d.lineup = moved
	return d, []Effect{
		CueTicker{ID: onAir.ID, Headline: onAir.Headline, Slot: onAir.Slot},
		Speak{ID: onAir.ID, Slot: onAir.Slot, Script: onAir.Script},
	}, false
}

// readInstead drops every stale card and puts the notice on the air in its
// place, so the listener hears WHY the read they were waiting for did not come.
//
// The notice is queued and the air taken in the same step. Queuing it and
// leaving it for the next event would put the gap it exists to explain in front
// of the explanation.
func (d Director) readInstead() (Director, bool) {
	d, notice, dropped := d.dropStale()
	if !dropped {
		return d, false
	}
	// ADMITTED BEFORE QUEUED. The lineup holds admitted cards only, so a
	// proposal handed to Queue is refused — and the refusal is silent, leaving
	// the listener with the gap AND no explanation, which is strictly worse
	// than the stale read this replaces.
	admitted, err := notice.To(Admitted)
	if err != nil {
		return d, false
	}
	queued, err := d.lineup.Queue(MainTrack, admitted)
	if err != nil {
		return d, false
	}
	d.lineup = queued
	standby, err := admitted.To(Standby)
	if err != nil {
		return d, false
	}
	if err := invariant.Check(!standby.Script.Empty(), "the notice reaches the air carrying its own words"); err != nil {
		return d, false
	}
	set, err := d.lineup.Set(standby)
	if err != nil {
		return d, false
	}
	d.lineup = set
	return d, true
}

// prepareNext gets the schedule ready ahead of the air, describing AT MOST ONE
// build.
//
// ONE AHEAD, AND ONLY ONE. A card moves to standby as its build is described,
// so the state transition is the guard against building it twice: nothing
// counts, and nothing remembers.
//
// A CARD WHOSE WORDS ARE FIXED AT PROPOSAL NEEDS NO BUILD (DR-7) — a burst
// head, a transition, the divert notice — so it is promoted and the card behind
// it considered. Stopping at one would leave a burst that opens with a head AND
// a transition with its first alert unprepared, and that alert's 1.03 s build
// would land as dead air instead of being ready when its turn came.
//
// A card left at ADMITTED is offered by Next for ever — only a standby card can
// take the air — and every card behind it goes unread, which is DR-3's
// guarantee failing from the other side.
func (d Director) prepareNext() (Director, []Effect) {
	// Bounded by the schedule: each pass promotes exactly one card out of
	// ADMITTED, and no card ever returns to it (P10-02).
	for range d.lineup.held() {
		next, track, ok := d.lineup.toPrepare()
		if !ok {
			return d, nil
		}
		// A build costs 1.03 s of network, and a stopped programme has no
		// cutover for it to be ready for — the report would only be stale when
		// one came.
		if !d.advances(track) {
			return d, nil
		}
		standby, err := next.To(Standby)
		if err != nil {
			return d, nil
		}
		if err := invariant.Check(standby.State == Standby, "the card reaches standby before its build is described"); err != nil {
			return d, nil
		}
		moved, err := d.lineup.Set(standby)
		if err != nil {
			return d, nil
		}
		// THE WALK MAKES PROGRESS, which is what bounds it: the card it just
		// looked at has left ADMITTED, so the next pass looks at a different
		// one. Without that this loop would offer the same card to itself
		// until the bound ran out, describing nothing.
		if err := invariant.Check(moved.held() == d.lineup.held(), "promoting a card neither adds one nor drops one"); err != nil {
			return d, nil
		}
		d.lineup = moved
		if !standby.Slot.textAtStandby() {
			// Propose refuses such a card without its words, so there is
			// nothing to build and nothing to wait for.
			if err := invariant.Check(!standby.Script.Empty(), "a card whose words were fixed at proposal reaches standby carrying them"); err != nil {
				return d, nil
			}
			continue
		}
		if err := invariant.Check(standby.Script.Empty(), "a card waiting to be built has no words yet"); err != nil {
			return d, nil
		}
		return d, []Effect{BuildCard{ID: standby.ID, Slot: standby.Slot, Subject: standby.Subject,
			Refs: standby.Refs, Divert: standby.Divert}}
	}
	return d, nil
}

// find is the card the lineup holds under this identity, if it still holds one.
// A completion for a card that has gone is ordinary, not an error: it was
// discarded or superseded while its work was in flight.
func (d Director) find(id string) (Card, bool) {
	if id == "" {
		return Card{}, false
	}
	for _, t := range []Track{AlertRail, MainTrack} {
		for _, c := range d.lineup.Cards(t) { // bounded by the track (P10-02)
			if c.ID == id {
				return c, true
			}
		}
	}
	return Card{}, false
}

// DescribeEvent is Describe for the other direction: what the Director was
// TOLD, in the same shape as what it decided.
//
// ONE OWNER FOR THIS FORMAT TOO, and for the same reason (DR-23). A timeline
// with the events in one style and the effects in another is two logs
// interleaved, and the whole point of the requirement is that a single log
// reads as a single story.
func DescribeEvent(ev Event) string {
	if err := invariant.Check(ev != nil, "an event describes something"); err != nil {
		return ""
	}
	switch v := ev.(type) {
	case Arrived:
		return fmt.Sprintf("arrived(%d)", len(v.Arrivals))
	case Tick:
		return "tick"
	case Built:
		return named("built", v.ID)
	case Finished:
		return named("finished", v.ID)
	case Failed:
		return named("failed", v.ID) + ":" + v.Reason
	case Powered:
		return named("powered", v.To.String())
	case Tuned:
		return named("tuned", v.Ref)
	case Programme:
		return fmt.Sprintf("programme(%d, %s)", len(v.Watchlist), v.Dwell)
	case Ended:
		return "ended"
	}
	return ""
}

// Trace is the line a card's state makes in the timeline: every card the
// schedule holds and what state it is in.
//
// A CARD'S TRANSITIONS ARE NOT LOGGED WHERE THEY HAPPEN, because where they
// happen is a pure function that may not write anything (Approach C). They are
// logged from the schedule they produce instead — which is strictly better than
// a line per transition, because it cannot disagree with the schedule it claims
// to describe.
func (l Lineup) Trace() string {
	var b strings.Builder
	for t := Track(0); t < numTracks; t++ { // bounded by the registry (P10-02)
		cards := l.Cards(t)
		if len(cards) == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(t.String() + "=[")
		for i, c := range cards { // bounded by the track (P10-02)
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(c.ID + ":" + c.State.String())
		}
		b.WriteString("]")
	}
	if b.Len() == 0 {
		return "empty"
	}
	return b.String()
}
