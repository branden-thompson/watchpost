import pathlib
# 0.19.0 D-180: the centre is answered in a box over the map.
# Killed by TestTheCentreIsAnsweredAsTheMapPans.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = 'mm.withPropCentre(mm.withDescriptionBox(mm.mapPane.lines, size, e), size, e)'
assert old in s, 'm214'
p.write_text(s.replace(old, 'mm.withDescriptionBox(mm.mapPane.lines, size, e)'))
