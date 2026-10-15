import pathlib
# 0.19.0 D-195: a transmitter at a watched place is its own record.
# Killed by TestTheStationIsTheWatchlistsOwnRecord.
p = pathlib.Path('modes/tty/map_role.go'); s = p.read_text()
old = 'l.Lat == t.Lat && l.Lon == t.Lon {'
assert old in s, 'm256'
p.write_text(s.replace(old, 'false {'))
