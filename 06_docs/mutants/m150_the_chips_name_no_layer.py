import pathlib
# 0.19.0 W2.4 (FR-1.14): the chips say the layer drawn.
# Killed by TestEachControlsStateIsSaid.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\t\tc.act, c.name = actMapLayer, propLayerName(d.mapPane.propLayer)'
assert old in s, 'm150'
p.write_text(s.replace(old, '\t\t\tc.act, c.name = actMapLayer, "Layer"'))
