import pathlib
# 0.19.6 B4.3 (design MS-1's S7): what the map reads next in the same call is
# the scroll it was lent, not the one it returned.
# Killed by TestPgDnKeepsTheMenusFocusInView.
p = pathlib.Path('modes/tty/map_model.go'); s = p.read_text()
old = '\tif w.ScrollSet {\n\t\te.Scroll = w.Scroll\n\t}\n\treturn e\n'
assert s.count(old) == 1, 'mMS6'
p.write_text(s.replace(old, '\treturn e\n', 1))
