import pathlib
# The scroll control goes back to spanning both tables — claiming a scroll the
# running order never does, and putting the pool's own headings inside the window
# it draws, so they vanish the moment the operator moves down the list (D-106).
p = pathlib.Path("modes/tty/broadcaster_lineup.go"); s = p.read_text()
old = """	return scrollSpan{lines: lines, from: -1, off: off, total: total}"""
assert old in s, "mAC1"
p.write_text(s.replace(old, """	return scrollSpan{lines: lines, from: 0, off: off, total: total}""", 1))
