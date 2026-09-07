import pathlib
# The positive test rewritten as a negative one — the shape that grows a hole
# every time a state is added. A card ON AIR is offered again, and so is a
# discarded one.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = "\t\t\tif c.State == Admitted || c.State == Standby {"
new = "\t\t\tif c.State != Done {"
assert old in s, "m59"
p.write_text(s.replace(old, new, 1))
