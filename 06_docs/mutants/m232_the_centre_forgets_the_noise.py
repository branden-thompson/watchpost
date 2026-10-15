import pathlib
# 0.19.0 D-187: the centre's key carries the snapshot's noise.
# Killed by TestTheChartIsAskedAnewForANewNoise.
p = pathlib.Path('modes/tty/map_prop_centre.go'); s = p.read_text()
old = 'radiusKm: key.radiusKm, noise: key.noise}, true'
assert old in s, 'm232'
p.write_text(s.replace(old, 'radiusKm: key.radiusKm}, true'))
