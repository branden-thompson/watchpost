// Package closedset walks a closed set and fails on the two ways a member goes
// unchecked.
//
// WHY THIS EXISTS. A producer emits members of a closed set; somewhere a
// consumer has to know every one of them. Nothing in Go makes that true, so it
// is written by hand — and by the third hand-written instance the discipline had
// already drifted: the first one, written for #15 at 0.14.2, forgot to catch a
// stale exemption, which the two written after it remembered. That is the drift
// this package removes.
//
// THE TWO BAD QUADRANTS, and they are the whole idea:
//
//	carried  &  declared absent  -> the exemption is STALE. Delete the row, or
//	                                the next member to regress looks expected.
//	!carried & !declared absent  -> nobody checked it, and nobody said why.
//
// A member is never SKIPPED. F-30's guard skips an unfixtured window and the
// 0.14.2 lane guard `continue`d past three lanes; both reported green while
// measuring less than they claimed. Skipping is how a set gate lies.
package closedset

import "testing"

// EachMember crosses every member against the declared-absent set.
//
// carried reports whether the consumer covers m, and may assert whatever else
// that member owes — it runs inside the walk, so its failures are attributed to
// the member. absent maps a member to the REASON no consumer can carry it; a
// reason is not a silencer, and a member that starts being carried while a
// reason still stands is itself a failure.
func EachMember[M comparable](t *testing.T, what string, members []M, absent map[M]string, carried func(M) bool) {
	t.Helper()
	if len(members) == 0 {
		t.Fatalf("%s: no members to check — the instrument cannot fail, so it proves nothing", what)
	}
	for _, m := range members {
		ok := carried(m)
		reason, declared := absent[m]
		switch {
		case ok && declared:
			t.Errorf("%s: %v is carried AND declared absent (%q) — delete the row, or the "+
				"next member to regress here will look expected", what, m, reason)
		case !ok && !declared:
			t.Errorf("%s: %v is in neither — carry it, or write down why nothing can", what, m)
		}
	}
}
