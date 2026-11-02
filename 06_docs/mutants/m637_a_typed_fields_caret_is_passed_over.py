import pathlib
# 0.19.3 D-362: in a field typed into, the cursor is on its caret.
# Killed by TestEveryFocusPointPlacesTheCursor.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = "\tfor i := len(lines) - 1; typing && i >= 0; i-- {\n"
assert s.count(old) == 1, 'm637'
p.write_text(s.replace(old, "\tfor i := len(lines) - 1; false && typing && i >= 0; i-- {\n", 1))
