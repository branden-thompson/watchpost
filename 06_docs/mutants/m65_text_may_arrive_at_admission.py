import pathlib
# DR-7 relaxed by one state — the report is composed when it is queued, so it
# says what the weather was then, not when it plays.
p = pathlib.Path("platform/lineup/card.go"); s = p.read_text()
old = 'invariant.Check(c.State == Standby, "a report\'s words materialise at standby")'
new = 'invariant.Check(c.State == Standby || c.State == Admitted, "a report\'s words materialise at standby")'
assert old in s, "m65"
p.write_text(s.replace(old, new, 1))
