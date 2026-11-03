import pathlib
# 0.19.3 D-362: in Settings the cursor is on the focused row's field caret.
# Killed by TestEveryFocusPointPlacesTheCursor.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = "\t\tif cc, typed := caretIn(lines[i], o.Glyphs().Cursor); typed {\n"
assert s.count(old) == 1, 'm638'
p.write_text(s.replace(old, "\t\tif cc, typed := caretIn(lines[i], o.Glyphs().Cursor); typed && false {\n", 1))
