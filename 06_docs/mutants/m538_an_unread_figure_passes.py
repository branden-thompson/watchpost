import pathlib
# 0.19.0 D-264: a density finding whose figure cannot be read passes instead of failing closed.
# Killed by TestARowUnderItsFloorFails.
p = pathlib.Path('tools/p10floor/main.go'); s = p.read_text()
old = '\t\tif mm == nil {\n\t\t\treturn 0, nil, fmt.Errorf("%s: the finding states no density: %q", f.File, f.Message)\n\t\t}\n'
assert s.count(old) == 1, 'm538'
p.write_text(s.replace(old, '\t\tif mm == nil {\n\t\t\tcontinue\n\t\t}\n', 1))
