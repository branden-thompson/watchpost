import pathlib
# 0.19.0 D-180: the centre is answered in a box over the map.
# Killed by TestTheCentreIsAnsweredAsTheMapPans.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = 'd.withPropCentre(d.withDescriptionBox(d.mapPane.lines, size), size)'
assert old in s, 'm214'
p.write_text(s.replace(old, 'd.withDescriptionBox(d.mapPane.lines, size)'))
