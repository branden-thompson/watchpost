import pathlib
# 0.19.0 FR-1.4: the Propagation layer is taken off as the mode is left.
# Killed by TestBothPropagationLayersDrawWithTheirUnits.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tif d.mapMode() == modeRadar || d.mapMode() == modeForecast || snap == nil || len(snap.Hours) == 0 {'
assert old in s, 'm148'
p.write_text(s.replace(old, '\tif snap == nil || len(snap.Hours) == 0 {'))
