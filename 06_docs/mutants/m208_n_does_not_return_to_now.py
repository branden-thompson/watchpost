import pathlib
# 0.19.0 FR-1.13: n returns to now.
# Killed by TestTheHoursStepOnAKey.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '\tcase actMapNewest:\n\t\td.mapPane.propAhead = 0\n'
assert old in s, 'm208'
p.write_text(s.replace(old, '\tcase actMapNewest:\n'))
