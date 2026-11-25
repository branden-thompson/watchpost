import pathlib
# 0.19.0 D-187: a snapshot for another noise is asked again.
# Killed by TestTheNoiseSettingIsTheUpdates.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'madeWith(*snap, mm.mapPane.propSnapCorrect) == mm.propOptions()'
assert s.count(old) == 1, 'm230'
p.write_text(s.replace(old, 'func() bool { m, o := madeWith(*snap, mm.mapPane.propSnapCorrect), mm.propOptions(); m.Noise = o.Noise; return m == o }()', 1))
