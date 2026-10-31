import pathlib
# 0.19.0 D-25, D-195: the tower is always marked.
# Killed by TestTheServiceAreaIsAlwaysDrawn.
p = pathlib.Path('modes/tty/map_zone.go'); s = p.read_text()
old = '\tif !d.mapRole().zone || loc == nil {\n\t\treturn tuimaps.Place{}, false\n'
assert old in s, 'm260'
p.write_text(s.replace(old, '\tif true {\n\t\treturn tuimaps.Place{}, false\n'))
