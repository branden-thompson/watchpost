import pathlib
# F-109 / FR-5.5, HUM LEAD ruling C (2026-09-16). The boundary never reaches the
# STATION AIR row, so a confident "*** ON AIR · BROADCASTING ***" stands with
# nothing qualifying it — and an operator who infers their antenna is radiating
# has been misled by us. That is D-153's dead end restored: the sentence assigned
# to a variable and rendered nowhere.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\t\t\tif render.Width(rung) <= max(0, room) {"
new = "\t\t\tif false {"
assert old in s, "mFR55"
p.write_text(s.replace(old, new, 1))
