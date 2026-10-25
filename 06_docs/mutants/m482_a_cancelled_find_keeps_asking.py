import pathlib
# 0.19.0 BUILD red team round 2 (FR-4.9, PN2-7): Esc over a find stops its lookup.
# Killed by TestACancelledFindCancelsItsRequest.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\tif d.mapPane.propFindCancel != nil {'
assert s.count(old) == 1, 'm482'
p.write_text(s.replace(old, '\tif false {', 1))
