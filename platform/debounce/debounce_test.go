package debounce

import (
	"testing"
	"time"
)

// THE ANSWER TO THE QUESTION THE OPERATOR HAS SINCE CHANGED MUST BE DROPPED.
// This is the whole reason the gate counts instead of cancelling: a resolve
// already in flight cannot be called back, so it has to be refused on arrival.
func TestAnAnswerForAnOlderEditIsRefused(t *testing.T) {
	var g Gate
	g = g.Edit() // "Rainbo"
	asked := g.Seq()
	g = g.Edit() // "Rainbow" — the first answer is now about the wrong text

	if g.Admits(asked) {
		t.Fatal("the gate admitted a pause for an edit that has been typed over")
	}
	out, took := g.Settle(asked)
	if took {
		t.Error("a stale answer was accepted")
	}
	if out.Settled() {
		t.Error("a refused answer must leave the field knowing nothing")
	}
}

func TestTheAnswerToTheLastEditIsTaken(t *testing.T) {
	var g Gate
	g = g.Edit().Edit().Edit()
	out, took := g.Settle(g.Seq())
	if !took || !out.Settled() {
		t.Fatalf("the current edit's answer must be taken: took=%v settled=%v", took, out.Settled())
	}
	// AND THE NEXT KEYSTROKE UNSETTLES IT. What the field knew was about the
	// text before the key, and drawing it afterwards is the stale-frame defect
	// one layer up from the memo.
	if out.Edit().Settled() {
		t.Error("an edit after an answer must clear it")
	}
}

// THE ZERO VALUE IS AN UNTOUCHED FIELD: nothing asked, nothing known.
func TestTheZeroGateKnowsNothing(t *testing.T) {
	var g Gate
	if g.Settled() {
		t.Error("an untouched field must not claim an answer")
	}
	if !g.Admits(0) {
		t.Error("the zero gate must admit its own sequence")
	}
}

// THE PAUSE IS A RULING, NOT A TUNING KNOB (HUM LEAD, 2026-09-14: "waits
// 300ms / *then* we can do a resolve check").
//
// PINNED BECAUSE ZERO IS THE DANGEROUS VALUE, not because 300 is magic. At zero
// the gate still works perfectly and every keystroke reaches the resolver —
// which for a location field is a geocoder call per key, the exact thing
// setup.go's AI-8 rule forbids. A floor is what makes that a test failure
// instead of a quiet ToS problem.
func TestThePauseIsLongEnoughToBeAPause(t *testing.T) {
	if Pause <= 0 {
		t.Fatal("a zero pause is no debounce at all: every keystroke asks")
	}
	if Pause < 100*time.Millisecond {
		t.Errorf("Pause is %v; below ~100ms ordinary typing reaches the resolver between keys", Pause)
	}
	if Pause > time.Second {
		t.Errorf("Pause is %v; above a second the field reads as broken rather than thoughtful", Pause)
	}
}
