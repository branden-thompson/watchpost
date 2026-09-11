import pathlib
# The duck asks only whether the BED carries, so nothing gives way for a report
# reading under the rail.
#
# The arbiter still dips per SEQUENCE and takes it back the moment nothing is
# waiting, so between two hazards the engine un-holds the report for the few
# milliseconds the schedule takes to dispatch the next one. On a rendered report
# that is not a volume bounce — the player is PAUSED, so what comes back is a
# fragment of a word. MVS-D-67 is "one duck per RAIL DRAIN, never per card", and
# the Duck/Restore pair is the only thing that can span a drain.
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
old = """\tif d.bed.carries {
\t\treturn true
\t}
\t_, reading := d.lineup.OnAir(MainTrack)
\treturn reading
"""
new = "\treturn d.bed.carries\n"
assert old in s, "mN4"
p.write_text(s.replace(old, new, 1))
