# 0.16.0 P3, N-3 / C-3. Delete the epoch check on the deck's report of a need.
#
# A NEED THAT ARRIVED AFTER THE LISTENER MOVED ON must not queue a card: they
# stopped the station, or tuned elsewhere, and would be read to anyway. The
# staleness rule guarded the AUDIO for three releases; the report is new surface
# and needs the same guard.
import pathlib
p = pathlib.Path("app/radio.go"); s = p.read_text()
old = "\tif !fresh {\n"
assert s.count(old) == 1, "m109"
p.write_text(s.replace(old, "\tif false && !fresh {\n"))
