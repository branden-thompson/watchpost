// Package closedset walks a closed set and fails on the two ways a member goes
// unchecked.
//
// WHY THIS EXISTS. A producer emits members of a closed set; somewhere a
// consumer has to know every one of them. Nothing in Go makes that true, so it
// is written by hand — and hand-written copies of the discipline drift: one
// catches a stale exemption, the next forgets to. One walk for every closed
// set removes that drift.
//
// THE TWO BAD QUADRANTS, and they are the whole idea:
//
//	carried  &  declared absent  -> the exemption is STALE. Delete the row, or
//	                                the next member to regress looks expected.
//	!carried & !declared absent  -> nobody checked it, and nobody said why.
//
// A member is never SKIPPED. A guard that skips an unfixtured window (F-30), or
// `continue`s past a lane, reports green while measuring less than it claims.
// Skipping is how a set gate lies.
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
