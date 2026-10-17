import pathlib
# 0.19.0 FR-1.10: the place found is marked.
# Killed by TestAFoundPlaceIsAnsweredAndNotKept.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\tif ref == nil {\n\t\treturn d\n\t}\n\tvar err error\n'
assert old in s, 'm387'
p.write_text(s.replace(old, '\tif true {\n\t\treturn d\n\t}\n\tvar err error\n'))
