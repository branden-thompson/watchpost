import pathlib
# D-159. `scopedMessage` claims every message it did not handle, so no key ever
# reaches `routeKey` or the active surface. The console stops answering the
# keyboard entirely — the F-72 shape, arrived at from the other end: there, the
# keymap was nil and the swap was unreachable; here, the message is consumed
# before the keymap is ever consulted.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	// NOT SCOPED TO ANYONE IN PARTICULAR: the key paths get their turn.
	return r, nil, false"""
new = """	// NOT SCOPED TO ANYONE IN PARTICULAR: the key paths get their turn.
	return r, nil, true"""
assert old in s, "mSC1"
p.write_text(s.replace(old, new, 1))
