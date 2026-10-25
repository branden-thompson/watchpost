import pathlib
# 0.19.0 P-1: an answer from or to the centre is asked once the view settles.
# Killed by TestAPanAsksOnceItSettles.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = 'func (d Dashboard) viewSettling() bool { return d.mapPane.viewGen != d.mapPane.viewAsked }'
assert s.count(old) == 1, 'm434'
p.write_text(s.replace(old, 'func (d Dashboard) viewSettling() bool { return false }', 1))
