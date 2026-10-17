import pathlib
# 0.19.0 FR-3.7: a mode opened again says the focus limit again.
# Killed by TestTheFocusLimitIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\td.mapPane.propSess = propSession{}\n'
assert old in s, 'm334'
p.write_text(s.replace(old, '\td.mapPane.propSess = propSession{applied: d.mapPane.propSess.applied}\n'))
