import pathlib
# 0.19.0 D-214: a snapshot asked with other stations is asked again.
# Killed by TestNoStationsIsASetting.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = ' && snap.Inputs.NoStations == d.mapPropNoGIRO) {'
assert old in s, 'm395'
p.write_text(s.replace(old, ') {'))
