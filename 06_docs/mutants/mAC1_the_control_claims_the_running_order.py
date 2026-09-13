import pathlib
# The scroll control goes back to spanning both tables — claiming a scroll the
# running order never does, and putting the pool's own headings inside the window
# it draws, so they vanish the moment the operator moves down the list (D-106).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	from := -1
	switch {
	case sched.from >= 0:
		from = sched.from
	case pool.from >= 0:
		from = len(sched.lines) + pool.from
	}"""
assert old in s, "mAC1"
p.write_text(s.replace(old, "	from := 0", 1))
