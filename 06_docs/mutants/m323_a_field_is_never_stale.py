import pathlib
# 0.19.0 FR-5.2: past an hour the field is said stale.
# Killed by TestAStaleFieldIsSaidStale.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif age > propStaleAfter {\n'
assert old in s, 'm323'
p.write_text(s.replace(old, '\tif false && age > propStaleAfter {\n'))
