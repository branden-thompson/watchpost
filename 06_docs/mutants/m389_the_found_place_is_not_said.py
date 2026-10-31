import pathlib
# 0.19.0 FR-1.10: the place found is said.
# Killed by TestAFoundPlaceIsAnsweredAndNotKept.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\treturn "Found " + f.Label + ": marked at the map\'s centre, and not kept."\n'
assert old in s, 'm389'
p.write_text(s.replace(old, '\treturn f.Label[:0]\n'))
