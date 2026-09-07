import pathlib
# The guard removed: a card reaches the air with nothing to say, which is
# silence where the ticker has already promised a callout.
p = pathlib.Path("platform/lineup/card.go"); s = p.read_text()
old = """	if err := invariant.Check(next != OnAir || c.Words() != "", "a card takes the air with its words already on it"); err != nil {
		return c, err
	}
"""
assert old in s, "m66"
p.write_text(s.replace(old, "", 1))
