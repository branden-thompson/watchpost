import pathlib
# 0.19.0 FR-1.10: focus returns to the map once a place is found.
# Killed by TestFocusReturnsAfterAFind.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\tif d.modal == modalPropFind {\n\t\td = d.close() // focus back on the map (FR-1.10)\n'
assert old in s, 'm385'
p.write_text(s.replace(old, '\tif false && d.modal == modalPropFind {\n\t\td = d.close() // focus back on the map (FR-1.10)\n'))
