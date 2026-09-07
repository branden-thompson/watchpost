package lineup

// fault.go — DR-21: failures reach the Director only when they change the
// schedule, through ONE escalation channel, surfaced by grade.
//
// THE DEFAULT IS TO ROUTE AROUND IT AND SAY NOTHING LOUD. A relay that dies
// falls through to the next mount and then to synth; a voice that cannot render
// falls back to another voice. Those heal themselves, and raising a modal for
// one would train a listener to dismiss the window that matters — a noise
// regression, which on a safety surface is a safety regression.
//
// A modal is for the fault that STOPS THE SCHEDULE: the one after which nothing
// is on the air and nothing is coming. That is the case a person has to know
// about, because nothing else is going to fix it.

import "github.com/branden-thompson/watchpost/platform/invariant"

// Escalate is the one escalation channel. The Director describes it; what a
// listener sees is the app's decision, and today that is the fault window.
//
// IT CARRIES THE REASON THE PRODUCER GAVE. A window that says "something went
// wrong" tells a listener what they already knew — that the station is quiet.
type Escalate struct {
	isEffect
	ID, Reason string
}

// stopped reports whether the schedule has nothing left to do: nothing on the
// air, and nothing held that could take it.
//
// IT IS ASKED AFTER THE FAULT HAS BEEN HANDLED, never before. The question is
// not "was this card important" — the Director cannot know that — but "is there
// anything left", and that is only answerable once the failed card is out and
// the schedule has settled around its absence.
// ONE CONDITION, NOT TWO. This began as "nothing on air AND nothing held", and
// the mutant that deleted the on-air half SURVIVED — correctly, because OnAir
// scans the very tracks held() counts, so an empty schedule cannot have a card
// on the air. The second check was the same rule written twice, which reads
// like extra safety and is extra surface: a later reader has to work out
// whether the two can disagree, and they cannot.
func (d Director) stopped() bool { return d.lineup.held() == 0 }

// escalation is the effect a fault raises, or nothing when the Director routed
// around it.
func (d Director) escalation(ev Failed) []Effect {
	// A DELIBERATE NON-DELIVERY IS NOT A FAULT, whatever the schedule looks
	// like afterwards (I-2). This is asked FIRST because stopped() cannot tell
	// the two apart: a burst is one card, so the schedule is empty after any
	// rail card leaves and the emptiness says nothing about why.
	if ev.Routed {
		return nil
	}
	if !d.stopped() {
		return nil // something else is on the air or waiting: it was routed around
	}
	if err := invariant.Check(ev.ID != "", "an escalation names the card that failed"); err != nil {
		return nil
	}
	reason := ev.Reason
	if reason == "" {
		reason = "the card could not be delivered"
	}
	return []Effect{Escalate{ID: ev.ID, Reason: reason}}
}
