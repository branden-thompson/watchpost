import pathlib
# 0.19.0 L-F2: an hour ahead is its own snapshot, with none of now's disturbance.
# Killed by TestAnHourAheadCarriesNoDisturbance.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '\t\treturn ahead\n'
assert old in s, 'm393'
p.write_text(s.replace(old, '\t\tahead.Disturbance = snap.Disturbance\n\t\treturn ahead\n'))
