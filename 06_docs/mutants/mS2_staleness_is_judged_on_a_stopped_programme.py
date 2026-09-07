import pathlib
# Staleness is judged ABOVE the power gate rather than below it.
#
# A card sitting in standby while the listener has stopped the programme is not
# stale — it is WAITING. Judging it early discards a schedule nobody abandoned,
# and the listener resumes to a notice explaining that reports they never heard
# have been dropped. The ORDER of these two guards is the rule (PD-3).
#
# Anchored on the two blocks as literal text rather than by index arithmetic:
# the first version located the stale block by searching for a call inside it,
# and went stale itself the moment that call was restructured to remove a
# recursion. An anchor that names what it swaps survives what an anchor that
# navigates to it does not.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()

gate = """	if !d.advances(track) {
		return d, nil, false // the listener stopped the programme (PD-1)
	}
"""
stale = """	// PD-3, AND ONLY HERE. Staleness is a question about a card that is about
	// to be READ. A card sitting in standby while the programme is stopped is
	// not stale, it is waiting — which is why this sits below the advances
	// guard and not on a tick.
	if d.now.Sub(next.BuiltAt) > StaleAfter && !next.BuiltAt.IsZero() {
		d, queued := d.readInstead()
		return d, nil, queued
	}
"""
assert gate + stale in s, "mS2"
p.write_text(s.replace(gate + stale, stale + gate, 1))
