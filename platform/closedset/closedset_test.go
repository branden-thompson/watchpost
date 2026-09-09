package closedset

import "testing"

// TestEachMemberFailsOnBothBadQuadrants is the instrument's own validation. A
// gate nobody has watched fail is not a gate, and this one exists precisely
// because two earlier hand-written gates reported green while measuring less
// than they claimed.
func TestEachMemberFailsOnBothBadQuadrants(t *testing.T) {
	for _, tc := range []struct {
		name    string
		carried map[string]bool
		absent  map[string]string
		want    int
	}{
		{"all carried, none declared", map[string]bool{"a": true, "b": true}, nil, 0},
		{"one declared, not carried", map[string]bool{"a": true}, map[string]string{"b": "cannot exist"}, 0},
		{"MISSING — carried by nobody, declared by nobody", map[string]bool{"a": true}, nil, 1},
		{"STALE — carried and still declared absent", map[string]bool{"a": true, "b": true}, map[string]string{"b": "cannot exist"}, 1},
		{"both at once", map[string]bool{"b": true}, map[string]string{"b": "stale"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spy := &testing.T{}
			EachMember(spy, "probe", []string{"a", "b"}, tc.absent, func(m string) bool { return tc.carried[m] })
			if got := spy.Failed(); (tc.want > 0) != got {
				t.Errorf("failed=%v, want failure=%v", got, tc.want > 0)
			}
		})
	}
}

// TestAnEmptySetIsARefusal — a walk over nothing passes trivially, which is the
// shape every one of this repo's false-green gates had.
func TestAnEmptySetIsARefusal(t *testing.T) {
	spy := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		EachMember(spy, "probe", []string{}, nil, func(string) bool { return true })
	}()
	<-done
	if !spy.Failed() {
		t.Error("an empty member set must fail: an instrument that cannot fail proves nothing")
	}
}
