package lineup

import (
	"testing"
	"time"
)

// A CARD THAT FAILED IS NOT RE-ADMITTED ON THE VERY NEXT OFFER.
//
// THE UAT DEFECT (HUM LEAD, 2026-09-10): "the lineup is FLYING through
// locations rapidly even on standby — so it seems like cards are constantly
// getting discarded and proposed / accepted."
//
// It was, and the loop is tight and unthrottled. `decline` returns
// `Failed{Routed: true}` on every refusal an executor can make — the listener is
// muted, the report could not be composed, no script rendered — the Director
// discards the card and PUBLISHES, the publish executor answers a publish by
// asking the producer to top the line-up off, the producer offers the same
// watchlist it always offers, and `ReadID` gives the failed location the very
// same identity it had a microsecond ago. Admit, build, fail, discard, publish,
// offer, admit. At pump speed, for as long as the fault lasts.
//
// The executor's own comment says the alerts "will be offered again" on the
// producer's "next cycle". Nothing was wrong with that intent; what was missing
// is that a publish IS a cycle, so "next" meant "now".
func TestAFailedCardSitsOutBeforeItIsOfferedAgain(t *testing.T) {
	d := offering(t, 3, "bonsall", "oceanside", "vista")
	d, _ = d.Step(offers("bonsall", "oceanside", "vista"))
	cards := d.lineup.Cards(MainTrack)
	if len(cards) != 3 {
		t.Fatalf("precondition: three cards admitted, got %d", len(cards))
	}
	failed := cards[0]
	d, _ = d.Step(Failed{ID: failed.ID, Reason: "the report could not be composed", Routed: true})
	if got := subjects(d); len(got) != 2 {
		t.Fatalf("the failed card leaves the schedule: %v", got)
	}

	// THE VERY NEXT OFFER — which in production is the publish that failure
	// itself caused — must not put it straight back.
	d, _ = d.Step(offers("bonsall", "oceanside", "vista"))
	for _, s := range subjects(d) {
		if s == failed.Subject {
			t.Fatalf("%q was re-admitted on the next offer: %v", failed.Subject, subjects(d))
		}
	}

	// AND IT COMES BACK. A fault that clears must not silence a location for the
	// rest of the run — the cool-off is a rate limit, not a blacklist.
	d, _ = d.Step(Tick{Now: d.now.Add(retryAfter + time.Second)})
	d, _ = d.Step(offers("bonsall", "oceanside", "vista"))
	found := false
	for _, s := range subjects(d) {
		found = found || s == failed.Subject
	}
	if !found {
		t.Errorf("after the cool-off %q is offered again: %v", failed.Subject, subjects(d))
	}
}

// THE MEMORY CANNOT GROW. It is the same concern D-48 was ruled on — "read
// history is too overweight" — and the same answer: a fixed size, oldest
// evicted, nothing to cap or own at runtime.
func TestTheDeclineMemoryIsBounded(t *testing.T) {
	d := offering(t, 1, "bonsall")
	for i := range retryMemory * 3 {
		d = d.noteDeclined(string(rune('a'+i%26)) + string(rune('a'+i/26)))
	}
	if got := len(d.declined); got > retryMemory {
		t.Errorf("the decline memory holds %d refs; it is bounded at %d", got, retryMemory)
	}
}
