import pathlib
# 0.19.0 D-272: a status D-247 does not name is noted at every ask, flooding the diagnostics.
# Killed by TestAnUnnamedStatusIsNotedOnce.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\tfirst := !f.seen[key]\n'
assert s.count(old) == 1, 'm559'
p.write_text(s.replace(old, '\tfirst := true\n', 1))
