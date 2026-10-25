import pathlib
# 0.19.0 D-187: the chart's key carries the snapshot's options, its noise among them.
# Killed by TestTheChartIsAskedAnewForANewNoise.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = 'made: d.propMadeOf(shown),'
assert s.count(old) == 1, 'm231'
p.write_text(s.replace(old, 'made: propMade{computed: shown.Computed, built: d.mapPane.propBuilt},', 1))
