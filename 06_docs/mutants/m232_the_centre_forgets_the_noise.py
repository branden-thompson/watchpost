import pathlib
# 0.19.0 D-187: the centre's key carries the snapshot's noise.
# Killed by TestTheChartIsAskedAnewForANewNoise.
p = pathlib.Path('modes/tty/map_prop_centre.go'); s = p.read_text()
old = 'propCentreKey{propChartKey: key, centre: centre}, true'
assert old in s, 'm232'
p.write_text(s.replace(old, 'propCentreKey{propChartKey: propChartKey{computed: key.computed, hour: key.hour, origin: key.origin, radiusKm: key.radiusKm}, centre: centre}, true'))
