import pathlib
# An open question stops owning the keyboard, so a digit typed into the position
# field reaches the running order and opens a different card underneath — and a
# stray key over the drop confirmation reaches the console (D-118, D-58 one
# window deeper).
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	switch r.observer.cardAct {
	case cardActionMove:
		return r.moveWindowKey(k)
	case cardActionDrop:
		return r.dropWindowKey(k)
	}"""
assert old in s, "mAL4"
p.write_text(s.replace(old, "", 1))
