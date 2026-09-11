import pathlib
# A report's read claims the bed again, so it rides the pump's ONE lane.
#
# Every effect naming a shared output is serialised there, in order — right for
# the rail, whose cards are read one after another and whose duck must not be
# overtaken. A report's read BLOCKS for as long as the words take, so the hazard
# the Director has just admitted cannot even be ASKED FOR until the weather
# finishes: the console shows the takeover and the listener hears it minutes
# later. The schedule's half of pre-emption is worthless without this half.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\tcase Speak:
\t\tif v.Track == AlertRail {
\t\t\tout = append(out, TheBed)
\t\t}
"""
new = """\tcase Speak:
\t\t_ = v
\t\tout = append(out, TheBed)
"""
assert old in s, "mN2"
p.write_text(s.replace(old, new, 1))
