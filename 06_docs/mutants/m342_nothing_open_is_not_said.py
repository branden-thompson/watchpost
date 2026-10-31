import pathlib
# 0.19.0 D-127: a target with no band open says so first.
# Killed by TestTheBandsAreSaidOpenFirst.
p = pathlib.Path('platform/propanswers/words.go'); s = p.read_text()
old = '\tif !strings.HasPrefix(parts[0], "open on ") && !strings.HasPrefix(parts[0], "disturbed on ") {\n'
assert old in s, 'm342'
p.write_text(s.replace(old, '\tif false {\n'))
