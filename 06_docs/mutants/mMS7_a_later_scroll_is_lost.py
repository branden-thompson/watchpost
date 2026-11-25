import pathlib
# 0.19.6 B4.3 (design MS-1's S7): two asks for the scroll in one call keep
# the first, though the code that asked wrote the second after it.
# Killed by TestPgDnKeepsTheMenusFocusInView.
p = pathlib.Path('modes/tty/map_model.go'); s = p.read_text()
old = '\tif v.ScrollSet {\n\t\tw.Scroll, w.ScrollSet = v.Scroll, true\n\t}\n'
assert s.count(old) == 1, 'mMS7'
p.write_text(s.replace(old, '\tif v.ScrollSet && !w.ScrollSet {\n\t\tw.Scroll, w.ScrollSet = v.Scroll, true\n\t}\n', 1))
