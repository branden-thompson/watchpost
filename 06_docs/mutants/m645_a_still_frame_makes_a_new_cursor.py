import pathlib
# 0.19.3 D-366: a frame where nothing moved hands out the cursor held.
# Killed by TestACursorHandedOutIsNeverChanged.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = '\t\treturn s.held\n'
assert s.count(old) == 1, 'm645'
p.write_text(s.replace(old, '\t\treturn &tea.Cursor{Position: s.held.Position, Color: s.held.Color, Shape: s.held.Shape}\n', 1))
