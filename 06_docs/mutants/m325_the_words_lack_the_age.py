import pathlib
# 0.19.0 FR-5.1: the words say the field's age.
# Killed by TestTheFieldsAgeIsAlwaysSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'return snapshotWords(*snap, d.mapPane.propLayer) + " · " + age //'
assert old in s, 'm325'
p.write_text(s.replace(old, 'return snapshotWords(*snap, d.mapPane.propLayer) + age[:0] //'))
