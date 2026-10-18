import pathlib
# 0.19.0 BUILD red team round 2 (P-1, CN2-1): a settle tick under a window waits for the map, never dropped.
# Killed by TestAPanSettledUnderAWindowIsAnsweredOnReturn.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\tif d.modal != modalMap && d.mapShown() {'
assert old in s, 'm477'
p.write_text(s.replace(old, '\tif false {', 1))
