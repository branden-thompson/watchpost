import pathlib
# 0.19.0 D-198: the shades' key is under the band legend.
# Killed by TestEachRowShowsItsNearestStation.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\tpropShadeLegend(o.ASCII)) // D-198'
assert old in s, 'm278'
p.write_text(s.replace(old, '\t\t"") // D-198'))
