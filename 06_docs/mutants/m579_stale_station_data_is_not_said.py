import pathlib
# 0.19.0 D-316, D-319: a newest station reading over 3 hours old is not said as a best approximation.
# Killed by TestThinOrStaleStationsAreSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'age > stationsThinAfter {'
assert s.count(old) == 1, 'm579'
p.write_text(s.replace(old, 'age > 100*stationsThinAfter {', 1))
