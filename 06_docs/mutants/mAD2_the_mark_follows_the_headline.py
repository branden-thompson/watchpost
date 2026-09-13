import pathlib
# The fabricated-event mark goes BEHIND the headline, where it is the first thing
# a narrow rule drops — and the whole point of the mark is that it cannot be the
# part that is missing (D-110).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	return testEventMark + " " + title"""
assert old in s, "mAD2"
p.write_text(s.replace(old, """	return title + " " + testEventMark""", 1))
