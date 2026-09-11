import pathlib
# The refresh stops standing aside for the alert rail, so the Composer's idle
# moment is spent on a REPORT while a hazard behind the one on air is still
# waiting to be composed. Its build then queues behind the weather.
#
# "The rail is prepared first, not merely aired first" is property 5 of the merge
# test, and it was written because a plant that reversed the precedence SURVIVED.
p = pathlib.Path("platform/lineup/stale.go"); s = p.read_text()
old = "\tif len(d.lineup.tracks[AlertRail]) > 0 {\n\t\treturn d, nil\n\t}\n"
assert old in s, "mP4"
p.write_text(s.replace(old, "", 1))
