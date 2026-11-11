import pathlib
# 0.19.0 FR-3.7: a mode opened again is an opening, so it says the short names again.
# Killed by TestTheShortNamesAreSaidInFull.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\td.mapPane.propSess = propSession{}\n'
assert old in s, 'm334'
p.write_text(s.replace(old, '\td.mapPane.propSess = propSession{applied: d.mapPane.propSess.applied}\n'))
