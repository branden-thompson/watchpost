import pathlib
# The ON AIR lock deleted. The card being read aloud accepts a new script.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = """	edited := have.Words() != c.Words() || have.ReadBy != c.ReadBy || have.Max != c.Max ||
		have.Subject != c.Subject || have.Headline != c.Headline"""
assert old in s, "m60 edited"
s = s.replace(old, "", 1)
old2 = """	if err := invariant.Check(have.State != OnAir || !edited, "a card on the air refuses edits; only its state may move"); err != nil {
		return l, err
	}
"""
assert old2 in s, "m60 check"
p.write_text(s.replace(old2, "", 1))
