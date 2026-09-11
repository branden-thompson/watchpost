import pathlib
# The positive test rewritten as a negative one — the shape that grows a hole
# every time a state is added. A card ON AIR is offered again, and so is a
# discarded one.
#
# RE-ANCHORED AT D-75, which added the fence clause beside it. The rule this
# guards is the STATE test and nothing else, so the fence clause is carried
# through unchanged rather than mutated with it — a mutant that broke two rules
# at once would not say which one the tests caught.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = "\t\t\tif (c.State == Admitted || c.State == Standby) && !c.OutOfFence {"
new = "\t\t\tif c.State != Done && !c.OutOfFence {"
assert old in s, "m59"
p.write_text(s.replace(old, new, 1))
