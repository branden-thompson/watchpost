import pathlib
# 0.19.0 FR-1.10: focus returns to the map once a place is found.
# Killed by TestFocusReturnsAfterAFind.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\tif e.Window == mapModalPropFind {\n\t\treturn mm, nil, mapWant{Close: true, Then: mapStepFoundRecentre} // focus back on the map (FR-1.10), then the place\n'
assert old in s, 'm385'
p.write_text(s.replace(old, '\tif false && e.Window == mapModalPropFind {\n\t\treturn mm, nil, mapWant{Close: true, Then: mapStepFoundRecentre} // focus back on the map (FR-1.10), then the place\n'))
