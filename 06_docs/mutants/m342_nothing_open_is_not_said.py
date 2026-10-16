import pathlib
# 0.19.0 D-127: a target with no band open says so first.
# Killed by TestTheBandsAreSaidOpenFirst.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif bandsOpen(bands) == 0 {\n'
assert old in s, 'm342'
p.write_text(s.replace(old, '\tif bandsOpen(bands) < 0 {\n'))
