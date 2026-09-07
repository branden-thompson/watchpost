import pathlib
# The release restricted to one exit, which is exactly today's shape: read in
# full releases the band, everything else leaves it holding a stale callout.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "	wasOnAir := card.State == OnAir"
new = "	wasOnAir := card.State == OnAir && to == Done"
assert old in s, "m92"
p.write_text(s.replace(old, new, 1))
