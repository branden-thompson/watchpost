import pathlib
# The shared type-ahead writes the LISTENER's default location even when it is
# filling the station's transmitter — one control, two settings, and the wrong
# one saved. D-72 split those facts precisely so this could not happen (D-115).
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = """	if d.setup.focus == rowTransmitter {
		into, cur, next = &d.setup.txRef, d.currentTransmitter(), rowServiceRadius
	}"""
assert old in s, "mAI2"
p.write_text(s.replace(old, "", 1))
