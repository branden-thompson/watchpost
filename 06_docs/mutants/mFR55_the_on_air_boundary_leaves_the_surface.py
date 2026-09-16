import pathlib
# F-109 / FR-5.5, HUM LEAD 2026-09-16: the boundary gets a line of its own in the
# station section. Here it never reaches the surface, so a confident
# "*** ON AIR \u00b7 BROADCASTING ***" stands with nothing qualifying it — and an
# operator who infers their antenna is radiating has been misled by us. That is
# D-153's dead end restored: the sentence assigned and rendered nowhere.
#
# Re-pointed 2026-09-16: ruling C put it inline on the STATION AIR row, where it
# fitted only at 160 cells and above; the line of its own covers every drawable
# width.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\t\t\tif render.Width(words) <= lane {"
new = "\t\t\tif false {"
assert old in s, "mFR55"
p.write_text(s.replace(old, new, 1))
