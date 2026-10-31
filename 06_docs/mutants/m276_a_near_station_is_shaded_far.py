import pathlib
# 0.19.0 D-198: a station within 500 km is the darkest shade.
# Killed by TestEachRowShowsItsNearestStation.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tcase n.Km < propNearKm:\n'
assert old in s, 'm276'
p.write_text(s.replace(old, '\tcase n.Km < 0:\n'))
