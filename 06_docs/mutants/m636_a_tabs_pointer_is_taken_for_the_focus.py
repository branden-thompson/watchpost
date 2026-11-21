import pathlib
# 0.19.3 D-362: only a pointer with nothing but space and a box's side before it marks the focused row.
# Killed by TestEveryFocusPointPlacesTheCursor.
p = pathlib.Path('modes/tty/kit_cursor.go'); s = p.read_text()
old = "\t\tif !strings.ContainsRune(Sides, r) {\n\t\t\treturn 0, false\n\t\t}\n"
assert s.count(old) == 1, 'm636'
p.write_text(s.replace(old, "\t\tif !strings.ContainsRune(Sides, r) && false {\n\t\t\treturn 0, false\n\t\t}\n", 1))
