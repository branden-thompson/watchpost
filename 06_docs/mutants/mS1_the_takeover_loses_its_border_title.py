import pathlib
# The takeover's box stops naming itself in its border, so a hazard interrupting
# the programme is told apart from a location report only by reading the badge —
# and the shape that is supposed to be recognisable at a glance is gone.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\tif c.Slot == lineup.BreakingAlert {\n\t\treturn bcTakeoverTitle\n\t}\n"
assert old in s, "mS1"
p.write_text(s.replace(old, "", 1))
