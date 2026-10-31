import pathlib
# 0.19.0 W3.1, D-155: L draws foF2 in the place of MUF(3000).
# Killed by TestBothPropagationLayersDrawWithTheirUnits.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\td.mapPane.propLayer = 1 - d.mapPane.propLayer\n'
assert old in s, 'm146'
p.write_text(s.replace(old, '\td.mapPane.propLayer = d.mapPane.propLayer + 0\n'))
