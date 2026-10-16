import pathlib
# 0.19.0 D-136: the focus limit is said once on opening, not after a refresh.
# Killed by TestTheFocusLimitIsSaid.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif d.mapPane.propApplied < 2 {\n'
assert old in s, 'm330'
p.write_text(s.replace(old, '\tif d.mapPane.propApplied < 99 {\n'))
