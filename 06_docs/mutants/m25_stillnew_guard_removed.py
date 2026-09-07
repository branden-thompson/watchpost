import pathlib
# The already-read guard goes, so every cycle offers the same alerts again and
# the station reads the same hazards over and over.
#
# RE-ANCHORED AT T3.10b: the guard moved with startTakeover's rewrite — the slot
# and the deferred release it used to sit beside are gone, and the marking that
# feeds it now happens in the executors as each line is said.
p = pathlib.Path("app/ticker.go"); s = p.read_text()
old = """	fresh = unread(fresh, t.seen.set())
	if len(fresh) == 0 {
		return
	}
"""
assert old in s, "m25"
p.write_text(s.replace(old, "", 1))
