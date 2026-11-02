import pathlib
# 0.19.3 D-367: while the terminal is focused, the cursor's cell is tinted.
# Killed by TestAFocusedTerminalTintsTheCellUnderTheCursor.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = '\tif blurred || !papered {\n'
assert s.count(old) == 1, 'm649'
p.write_text(s.replace(old, '\tif true || blurred || !papered {\n', 1))
