import pathlib
# The pool stops reserving its own rows, so the running order takes every line it
# can and the pool draws in the remainder — the HUM LEAD's own report: "I can only
# see 12 locations of the 24 location pool" (D-104).
p = pathlib.Path("modes/tty/broadcaster_pool.go"); s = p.read_text()
old = """	return min(bcPoolRows, len(b.area.Pool)) + bcPoolChrome"""
assert old in s, "mAB2"
p.write_text(s.replace(old, "	return 0", 1))
