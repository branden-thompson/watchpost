import pathlib
# 0.19.0 AX-13: in the Propagation mode the night's edge alone is said.
# Killed by TestTheNightIsSaidAsItIsDrawn.
p = pathlib.Path('modes/tty/map_night.go'); s = p.read_text()
old = '\tif d.mapMode() == modePropagation {\n\t\treturn "The night side\'s edge'
assert old in s, 'm421'
p.write_text(s.replace(old, '\tif d.mapMode() == modePropagation && false {\n\t\treturn "The night side\'s edge'))
