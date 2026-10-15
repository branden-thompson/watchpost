import pathlib
# 0.19.0 D-187: the chart's key carries the snapshot's noise.
# Killed by TestTheChartIsAskedAnewForANewNoise.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = 'radiusKm: float64(d.propRadiusKm()), noise: shown.Noise}, true'
assert old in s, 'm231'
p.write_text(s.replace(old, 'radiusKm: float64(d.propRadiusKm())}, true'))
