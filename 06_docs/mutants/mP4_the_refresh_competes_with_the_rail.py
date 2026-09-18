import pathlib
# The refresh stops standing aside for the alert rail, so the Composer's idle
# moment is spent on a REPORT while a hazard behind the one on air is still
# waiting to be composed. Its build then queues behind the weather.
#
# "The rail is prepared first, not merely aired first" is property 5 of the merge
# test, and it was written because a plant that reversed the precedence SURVIVED.
p = pathlib.Path("platform/lineup/stale.go"); s = p.read_text()
# Re-pointed 2026-09-16 (D-150): the guard asks the PROJECTION now — what the
# rail can READ — because a rail of out-of-fence cards held it true for ever.
# The rule is unchanged: deleting the guard lets the refresh compete with the
# rail for the composer.
old = "\tif len(d.lineup.Projection(AlertRail)) > 0 {\n\t\treturn d, nil\n\t}\n"
assert old in s, "mP4"
p.write_text(s.replace(old, "", 1))
