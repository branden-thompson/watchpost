import pathlib
# 0.19.0 FR-3.6 (D-127): after a refresh the words say when they were updated.
# Killed by TestARefreshIsSaidOnce.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif d.mapPane.propApplied > 1 {\n'
assert old in s, 'm331'
p.write_text(s.replace(old, '\tif d.mapPane.propApplied > 99 {\n'))
