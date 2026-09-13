import pathlib
# The bed stops re-resolving when the station moves, so the selector goes on
# offering the relays of the region the station has LEFT — and tuning one of them
# points the transmitter at a stream for somewhere else entirely (D-117).
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = """	if ctx != nil {
		go lp.refreshBedRelays(ctx)
	}"""
assert old in s, "mAK4"
p.write_text(s.replace(old, "", 1))
