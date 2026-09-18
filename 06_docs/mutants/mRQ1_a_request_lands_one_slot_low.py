import pathlib
# D-156. The Line-Up Request window sends the typed SLOT where the schedule takes
# an INDEX. `Requested.To` is documented as "the same number `Moved.To` carries",
# and the move path has subtracted `liveOffset` since D-119 — so the two operator
# paths into one field disagree by one on STANDBY, the console's normal state.
# The card lands a row below the slot typed, silently, and the window's own
# confirmation names the slot that was asked for.
p = pathlib.Path("modes/tty/request.go"); s = p.read_text()
old = """	if n -= liveOffset; n < 0 {
		return 0
	}
	return n"""
assert old in s, "mRQ1"
p.write_text(s.replace(old, """	return n""", 1))
