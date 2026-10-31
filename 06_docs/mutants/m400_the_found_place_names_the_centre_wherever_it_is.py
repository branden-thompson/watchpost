import pathlib
# 0.19.0 AX-1: the found place names the centre only while the centre rests on it.
# Killed by TestTheCentreIsNamedWhereverItIs.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = '\tif f := d.mapPane.propFound; f != nil && propStepped(f.Lat, f.Lon) == c {\n'
assert old in s, 'm400'
p.write_text(s.replace(old, '\tif f := d.mapPane.propFound; f != nil {\n'))
