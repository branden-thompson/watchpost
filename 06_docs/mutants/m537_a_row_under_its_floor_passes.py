import pathlib
# 0.19.0 D-264: a density row measuring under its ratified floor passes.
# Killed by TestARowUnderItsFloorFails.
p = pathlib.Path('tools/p10floor/main.go'); s = p.read_text()
old = '\t\tif got < want {\n'
assert s.count(old) == 1, 'm537'
p.write_text(s.replace(old, '\t\tif got < want-1 {\n', 1))
