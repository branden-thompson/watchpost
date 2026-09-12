import pathlib
# The operator's selection wins even when there ISN'T one, so a station whose
# operator has never touched the selector reports an empty relay row instead of
# naming what the Director's bed is actually carrying. "Nothing chosen" and "the
# bed is on nothing" are different facts, and only one of them is true here.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = """	if x.selected != nil {
		if chosen := x.selected(); chosen != "" {
			return chosen
		}
	}"""
new = """	if x.selected != nil {
		return x.selected()
	}"""
assert old in s, "mY3"
p.write_text(s.replace(old, new, 1))
