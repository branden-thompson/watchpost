import pathlib
# F-109. The fit check goes, so the LONGEST rung is taken whatever the width and
# the row truncates it. A half-said safety statement is a different claim, not a
# shorter one — on the previous placement the cut rendered as "\u00b7 audio on" at 120
# cells: complete, reassuring, and the opposite of the sentence it came from.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """		for _, words := range bcAirBoundaries() { // bounded by the ladder (P10-02)
			if render.Width(words) <= lane {
				rows = append(rows, render.PadTo(render.Tint(words, render.Tok(render.AlertModalText)), lane))
				break
			}
		}"""
new = """		for _, words := range bcAirBoundaries() { // bounded by the ladder (P10-02)
			rows = append(rows, render.TruncateCells(render.Tint(words, render.Tok(render.AlertModalText)), lane))
			break
		}"""
assert old in s, "mFR56"
p.write_text(s.replace(old, new, 1))
