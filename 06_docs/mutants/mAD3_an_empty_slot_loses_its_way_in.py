import pathlib
# An undecided UP NEXT slot stops carrying its handle, so the operator can see the
# slot and cannot open it — F-97 exactly, one row along from where it was first
# filed (D-110).
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = """		return append(rows, b.cardControls(o, c, handle))
	}"""
assert old in s, "mAD3"
p.write_text(s.replace(old, """		return append(rows, "")
	}""", 1))
