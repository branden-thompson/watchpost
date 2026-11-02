import pathlib
# 0.19.3 D-362: the cursor is steady, a block that does not blink.
# Killed by TestTheCursorIsSteady.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = "\tc := &tea.Cursor{Position: tea.Position{X: x, Y: y}, Shape: tea.CursorBlock}\n"
assert s.count(old) == 1, 'm635'
p.write_text(s.replace(old, "\tc := &tea.Cursor{Position: tea.Position{X: x, Y: y}, Shape: tea.CursorBlock, Blink: true}\n", 1))
