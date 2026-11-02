import pathlib
# 0.19.3 D-366: a cursor handed out is never changed in place; Bubble Tea compares the next with it.
# Killed by TestACursorHandedOutIsNeverChanged.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = '\tc := &tea.Cursor{Position: tea.Position{X: x, Y: y}, Shape: tea.CursorBlock}\n'
assert s.count(old) == 1, 'm644'
p.write_text(s.replace(old, '\tc := &tea.Cursor{Position: tea.Position{X: x, Y: y}, Shape: tea.CursorBlock}\n\tif s != nil && s.held != nil {\n\t\t*s.held = *c\n\t\tc = s.held\n\t}\n', 1))
