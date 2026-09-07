import pathlib
# The pairing broken the other way: every card that leaves releases the band,
# including one that never took the air, clearing a callout it never made.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "	wasOnAir := card.State == OnAir"
new = "	wasOnAir := card.State != Proposed"
assert old in s, "m93"
p.write_text(s.replace(old, new, 1))
