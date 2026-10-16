import pathlib
# 0.19.0 FR-5.5: beyond 2000 km an answer is about as good as typical.
# Killed by TestEachRowShowsItsNearestStation.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\tcase n.Km >= propFarKm:\n'
assert old in s, 'm279'
p.write_text(s.replace(old, '\t\tcase false:\n'))
