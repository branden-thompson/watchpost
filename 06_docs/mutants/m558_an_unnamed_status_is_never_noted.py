import pathlib
# 0.19.0 D-272: a status D-247 does not name is never noted in the diagnostics.
# Killed by TestAnUnnamedStatusIsNotedOnce.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\tif namedStatus[status] {\n'
assert s.count(old) == 1, 'm558'
p.write_text(s.replace(old, '\tif namedStatus[status] || status > 0 {\n', 1))
