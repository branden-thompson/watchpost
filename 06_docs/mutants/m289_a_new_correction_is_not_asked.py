import pathlib
# 0.19.0 D-105: a snapshot asked without the correction is asked again.
# Killed by TestTheNoReadingsCorrectionIsASetting.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'madeWith(*snap, d.mapPane.propSnapCorrect) == d.propOptions()'
assert s.count(old) == 1, 'm289'
p.write_text(s.replace(old, 'func() bool { m, o := madeWith(*snap, d.mapPane.propSnapCorrect), d.propOptions(); m.CorrectNoReadings = o.CorrectNoReadings; return m == o }()', 1))
