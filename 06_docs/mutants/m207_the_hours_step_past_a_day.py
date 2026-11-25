import pathlib
# 0.19.0 FR-1.13: the hours ahead stop at 24.
# Killed by TestTheHoursStepOnAKey.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '\t\tmm.mapPane.propAhead = min(mm.mapPane.propAhead+1, propAheadMost)\n'
assert old in s, 'm207'
p.write_text(s.replace(old, '\t\tmm.mapPane.propAhead++\n'))
