import pathlib
# D-159. `routeKey` reports every key as handled, so the `switch r.active` below
# it never runs and neither surface sees a keystroke it did not have a rule for.
# The ordinary keys — everything that is not a binding, a slot digit or a window
# control — simply stop working.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	// NO KEY RULE CLAIMED IT: the active surface gets it.
	return r, nil, false"""
new = """	// NO KEY RULE CLAIMED IT: the active surface gets it.
	return r, nil, true"""
assert old in s, "mSC2"
p.write_text(s.replace(old, new, 1))
