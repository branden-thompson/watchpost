import pathlib
# The panic is contained and reported, but the card is never failed — so the
# schedule waits forever for a completion that will never come.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = """		id, ofCard := lineup.CardOf(f)
		if !ofCard {"""
new = """		id, ofCard := lineup.CardOf(f)
		if true || !ofCard {"""
assert old in s, "mB4"
p.write_text(s.replace(old, new, 1))
