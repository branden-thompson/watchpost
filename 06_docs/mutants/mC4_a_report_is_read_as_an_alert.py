import pathlib
# A location report's words are read through the narrator as one aside clip — the
# wrong path, quietly, instead of being declined.
#
# ITS ANCHOR WENT STALE and the gate did not notice until 2026-09-10: the guard
# it weakens kept its shape but the decline's WORDS changed when the two old
# checks became one, so the mutant stopped applying and reported as a failure to
# COMPILE rather than as a rule nobody was testing. A mutant is a test of a test;
# one that cannot be applied is neither.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\tif !onTheRail(v.Slot) {\n\t\treturn x.decline(v, v.ID, "no reader for this slot: the arbiter reads the rail, and the rail only")\n\t}\n'
new = '\tif !onTheRail(v.Slot) && v.Slot < 0 {\n\t\treturn x.decline(v, v.ID, "no reader for this slot: the arbiter reads the rail, and the rail only")\n\t}\n'
assert old in s, "mC4"
p.write_text(s.replace(old, new, 1))
