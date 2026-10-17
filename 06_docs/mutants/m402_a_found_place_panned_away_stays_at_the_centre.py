import pathlib
# 0.19.0 AX-1: a found place panned away from is marked on the map, not at its centre.
# Killed by TestTheCentreIsNamedWhereverItIs.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\tif c, ok := d.propCentreAt(); ok && propStepped(f.Lat, f.Lon) != c {\n'
assert old in s, 'm402'
p.write_text(s.replace(old, '\tif c, ok := d.propCentreAt(); ok && propStepped(f.Lat, f.Lon) != c && false {\n'))
