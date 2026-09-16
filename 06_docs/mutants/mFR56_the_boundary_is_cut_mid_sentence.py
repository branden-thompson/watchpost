import pathlib
# F-109. The fit check goes, so the longest rung is taken whatever the width and
# the row truncates it. Measured at 120 cells the operator reads
# "*** ON AIR · BROADCASTING *** · audio on" — a complete, reassuring phrase that
# says the opposite of the sentence it was cut from. A half-said safety statement
# is a different claim, not a shorter one.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """			if render.Width(rung) <= max(0, room) {
				air = rung
				break
			}"""
new = """			air = rung
			break"""
assert old in s, "mFR56"
p.write_text(s.replace(old, new, 1))
