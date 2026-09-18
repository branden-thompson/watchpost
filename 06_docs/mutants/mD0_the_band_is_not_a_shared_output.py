import pathlib
# The band stops being a resource, so a cue and a release ride their own
# goroutines. The release of the card LEAVING the air and the cue of the card
# TAKING it then race — and when the release lands second it clears the callout
# the cue just put up, so the next card reads with the band already back on
# rotation.
#
# RE-ANCHORED AT D-82: the band is claimed by the RAIL's cue and release, because
# a report neither writes the band nor should hold the lane while it reads. The
# rule is unchanged for the lane that has it.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\tcase CueTicker:
\t\tif v.Track == AlertRail {
\t\t\tout = append(out, TheBand)
\t\t}
\tcase ReleaseTicker:
\t\tif v.Track == AlertRail {
\t\t\tout = append(out, TheBand)
\t\t}
"""
assert old in s, "mD0"
p.write_text(s.replace(old, "\tcase CueTicker, ReleaseTicker:\n\t\t_ = v\n", 1))
