import pathlib
# The bed stops re-resolving when the station moves, so the selector goes on
# offering the relays of the region the station has LEFT — and tuning one of them
# points the transmitter at a stream for somewhere else entirely (D-117).
#
# THE GUARD IS NEUTERED RATHER THAN THE CALL DELETED: deleting it leaves `ctx`
# unused and the mutation does not compile, which is INVALID — no evidence either
# way — rather than a measurement.
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = """	if ctx != nil {
		go lp.refreshBedRelays(ctx)
	}"""
assert old in s, "mAK4"
p.write_text(s.replace(old, """	if ctx == nil {
		go lp.refreshBedRelays(ctx)
	}""", 1))
