import pathlib
# 0.19.0 BUILD red team round 2 (FR-4.9, PN2-7): Esc over a find stops its lookup.
# Killed by TestACancelledFindCancelsItsRequest.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\t\t\tif d.mapPane.propFindCancel != nil {'
assert old in s, 'm482'
p.write_text(s.replace(old, '\t\t\tif false {', 1))
