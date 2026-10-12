import pathlib
# 0.19.0 FR-1.9 (D-25, A-45): the chart answers for each continent.
# Killed by TestTheContinentsAreTheDefaultTargets.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tc.local, c.targets = local[0], paths\n'
assert old in s, 'm190'
p.write_text(s.replace(old, '\tc.local, c.targets = local[0], nil\n'))
