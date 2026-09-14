import pathlib
# The pool lookup falls through to the RESOLVER when the pool has no match — one
# network call per character typed, on a field the operator is typing into.
#
# HUM LEAD, 2026-09-14: "Whatever helps performance - the end result is
# transparent to the end user - either what they type is a valid location within
# the service radius or not." The pool alone answers that; the resolver bought
# only the difference between two helper sentences that both point at Observer.
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = """	// AND NOTHING ELSE IS ASKED."""
assert old in s, "mBC3"
new = """	if lp.resolverForTest != nil {
		if ref, _, err := lp.resolverForTest(query); err == nil {
			return ref, false, true
		}
	}
	// AND NOTHING ELSE IS ASKED."""
p.write_text(s.replace(old, new, 1))
