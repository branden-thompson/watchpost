import pathlib
# 0.19.3 D-366: a frame drawn again as it was hands out the cursor held.
# Killed by TestACursorHandedOutIsNeverChanged.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = '\t\treturn s.out, s.held\n'
assert s.count(old) == 1, 'm645'
p.write_text(s.replace(old, '\t\treturn s.out, s.look.cursor(x, y)\n', 1))
