import pathlib
# Half the identity invariant dropped. What is read changes without what the
# lineup says is read changing with it.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = "invariant.Check(have.Slot == c.Slot && have.Origin == c.Origin,"
new = "invariant.Check(have.Origin == c.Origin,"
assert old in s, "m67"
p.write_text(s.replace(old, new, 1))
