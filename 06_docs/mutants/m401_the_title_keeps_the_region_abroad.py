import pathlib
# 0.19.0 AX-1: away from every region the view is named by its centre, not the region last drawn.
# Killed by TestTheCentreIsNamedWhereverItIs.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = '\tif c, ok := d.propCentreAt(); ok && d.mapMode() == modePropagation {\n'
assert old in s, 'm401'
p.write_text(s.replace(old, '\tif c, ok := d.propCentreAt(); ok && d.mapMode() == modePropagation && false {\n'))
