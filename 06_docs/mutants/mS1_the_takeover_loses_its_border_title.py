import pathlib
# The takeover's box stops naming itself in its border, so a hazard interrupting
# the programme is told apart from a location report only by reading the badge —
# and the shape that is supposed to be recognisable at a glance is gone.
#
# RE-ANCHORED AT D-110, where EVERY card gained a border title: the takeover's is
# still its own, and dropping it now makes it read as one more report.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\t\treturn bcTakeoverTitle\n"
assert old in s, "mS1"
p.write_text(s.replace(old, "\t\treturn cardRuleTitle(c, g)\n", 1))
