import pathlib
# 0.19.3 D-367: the tint is 15 % of the glyph's colour.
# Killed by TestAFocusedTerminalTintsTheCellUnderTheCursor.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = 'const focusShare = 0.15\n'
assert s.count(old) == 1, 'm654'
p.write_text(s.replace(old, 'const focusShare = 0.5\n', 1))
