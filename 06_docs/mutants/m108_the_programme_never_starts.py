# 0.16.0 P3, and it is the shape of a blocker that shipped. Delete the deck's
# report that the programme is running.
#
# THE DIRECTOR BEGINS STOPPED, on purpose, so a station comes up silent. With
# nothing telling it otherwise, advances(MainTrack) is false for the life of the
# process: every card is refused, the lineup fills, and NO FAULT IS RAISED —
# because nothing failed and nothing was ever admitted. A permanently silent
# station is the hardest defect to see, and this is the wire that prevents it.
import pathlib
p = pathlib.Path("app/radio.go"); s = p.read_text()
old = "\td.tell(lineup.Monitored{Running: true})\n"
assert s.count(old) == 1, "m108"
p.write_text(s.replace(old, "\tif false {\n\t\td.tell(lineup.Monitored{Running: true})\n\t}\n"))
