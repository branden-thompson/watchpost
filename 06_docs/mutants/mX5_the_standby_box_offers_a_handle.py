import pathlib
# The standby box goes back to being drawn as a CARD, which is what it was before
# D-89: `lineup.Card{}`'s zero Slot IS a location report, so the box came out titled
# `LOCATION REPORT •STANDARD• [0]` — a report that does not exist, graded, with a
# chip that opens nothing. A chip on a slot with no card behind it is F-97 again.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = """				rows = append(rows, lane.standbyBox(bcReadCardRows)...)
				continue"""
new = """				c = lineup.Card{}"""
assert old in s, "mX5"
p.write_text(s.replace(old, new, 1))
