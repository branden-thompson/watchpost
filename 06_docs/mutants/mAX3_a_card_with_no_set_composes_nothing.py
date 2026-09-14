import pathlib
# A card that names no report set composes an EMPTY report instead of a full one.
# Every card in the tree is in that state until an operator requests otherwise —
# the Director's own cards carry no set — so the station would read a frame with
# nothing in it and the rotation would go quiet (R2).
p = pathlib.Path("app/schedule.go"); s = p.read_text()
old = """		want := report.Everything()
		if wants != nil {"""
new = """		var want report.Set
		if wants != nil {"""
assert old in s, "mAX3"
p.write_text(s.replace(old, new, 1))
