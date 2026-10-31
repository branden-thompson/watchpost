import pathlib
# 0.19.0 D-319 (R10-4): exactly half the stations reporting in 3 hours is said as too few.
# Killed by TestThinOrStaleStationsAreSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif in.StationsRecent*2 < in.StationsHeld {\n'
assert s.count(old) == 1, 'm580'
p.write_text(s.replace(old, '\tif in.StationsRecent*2 <= in.StationsHeld {\n', 1))
