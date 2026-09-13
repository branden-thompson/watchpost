import pathlib
# The bed stops re-resolving when the station moves, so the selector goes on
# offering the relays of the region the station has LEFT — and tuning one of them
# points the transmitter at a stream for somewhere else entirely (D-117).
#
# THE GUARD IS INVERTED RATHER THAN THE CALL DELETED: deleting it leaves `ctx`
# unused and the mutation does not compile, which is INVALID — no evidence either
# way — rather than a measurement.
#
# RE-ANCHORED AT D-120, when the call moved behind the `rebed` seam a test can
# hold. The rule is the same one; the mutation is one function along.
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = """	if ctx != nil {
		go lp.rebed(ctx)
	}"""
assert old in s, "mAK4"
p.write_text(s.replace(old, """	if ctx == nil {
		go lp.rebed(ctx)
	}""", 1))
