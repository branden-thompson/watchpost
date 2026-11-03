import pathlib
# 0.19.3 D-366: a cursor handed out is never changed in place; Bubble Tea compares the next with it.
# Killed by TestACursorHandedOutIsNeverChanged.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = '\tif held == nil || held.X != x || held.Y != y || s.look != look {\n\t\theld = look.cursor(x, y)\n\t}\n'
assert s.count(old) == 1, 'm644'
p.write_text(s.replace(old, '\tif held == nil {\n\t\theld = look.cursor(x, y)\n\t} else {\n\t\t*held = *look.cursor(x, y)\n\t}\n', 1))
