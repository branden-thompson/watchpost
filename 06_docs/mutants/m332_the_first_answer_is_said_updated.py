import pathlib
# 0.19.0 FR-3.6: the first answer is no refresh.
# Killed by TestARefreshIsSaidOnce.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif d.mapPane.propApplied > 1 {\n'
assert old in s, 'm332'
p.write_text(s.replace(old, '\tif d.mapPane.propApplied > 0 {\n'))
