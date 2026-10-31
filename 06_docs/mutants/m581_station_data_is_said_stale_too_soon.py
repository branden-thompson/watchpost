import pathlib
# 0.19.0 D-319 (R10-4): station data is said stale at 2 h 30 m, not 3 hours.
# Killed by TestThinOrStaleStationsAreSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'const stationsThinAfter = 3 * time.Hour\n'
assert s.count(old) == 1, 'm581'
p.write_text(s.replace(old, 'const stationsThinAfter = 150 * time.Minute\n', 1))
