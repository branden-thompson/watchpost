import pathlib
# A completion for a card the schedule no longer holds settles anyway, so a
# stranger\'s event publishes — telling every reader to re-read a lineup that did
# not change.
#
# REWRITTEN: the first form deleted a USE and left `left` unused, so the tree
# did not compile and the run reported INVALID — not evidence either way. A
# deletion mutant makes the rule UNREACHABLE and keeps the identifiers that hold
# the tree together.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\tif !left {\n"
new = "\tif !left && left {\n"
assert old in s, "mDB"
p.write_text(s.replace(old, new, 1))
