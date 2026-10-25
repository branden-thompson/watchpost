import pathlib
# 0.19.0 D-214: a snapshot asked with other stations is asked again.
# Killed by TestNoStationsIsASetting.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'madeWith(*snap, d.mapPane.propSnapCorrect) == d.propOptions()'
assert s.count(old) == 1, 'm395'
p.write_text(s.replace(old, 'func() bool { m, o := madeWith(*snap, d.mapPane.propSnapCorrect), d.propOptions(); m.NoStations = o.NoStations; return m == o }()', 1))
