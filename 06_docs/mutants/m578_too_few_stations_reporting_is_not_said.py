import pathlib
# 0.19.0 D-316, D-319: fewer than half the stations reporting in 3 hours is not said as a best approximation.
# Killed by TestThinOrStaleStationsAreSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif in.StationsRecent*2 < in.StationsHeld {\n'
assert s.count(old) == 1, 'm578'
p.write_text(s.replace(old, '\tif in.StationsRecent < 0 {\n', 1))
