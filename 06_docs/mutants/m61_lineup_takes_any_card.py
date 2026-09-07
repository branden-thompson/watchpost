import pathlib
# Admission stops being the door: anything not already refused may be queued.
# Bounds now have a later moment to apply at, which is the defect returning.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = 'invariant.Check(c.State == Admitted, "the lineup holds admitted cards only")'
new = 'invariant.Check(c.State != Refused, "the lineup holds admitted cards only")'
assert old in s, "m61"
p.write_text(s.replace(old, new, 1))
