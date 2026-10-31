import pathlib
# 0.19.0 FR-5.1: the age is the oldest input's, never the time computed.
# Killed by TestTheAgeIsTheOldestInputs.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tage = max(d.now().Sub(oldest), 0)\n'
assert old in s, 'm322'
p.write_text(s.replace(old, '\tage = max(d.now().Sub(s.Computed), oldest.Sub(oldest))\n'))
