import pathlib
# 0.19.0 W0.2: each propagation host's cap is its source's bound.
# Killed by TestEachPropagationSourceHasItsOwnCap.
p = pathlib.Path('app/mapprop.go'); s = p.read_text()
old = '\t\t\tcaps[h] = int64(s.MaxBytes)\n'
assert old in s, 'm320'
p.write_text(s.replace(old, '\t\t\tcaps[h] = 0\n'))
