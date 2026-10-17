import pathlib
# 0.19.0 I-6: a listed host on another port, or with credentials, is not asked.
# Killed by TestTheClientRefusesAnUnlistedHost.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\tif u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {\n'
assert old in s, 'm433'
p.write_text(s.replace(old, '\tif u.Scheme != "https" {\n'))
