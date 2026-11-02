import pathlib
# 0.19.3 D-366: the cursor wears the colour of the cell it covers.
# Killed by TestTheCursorWearsTheColourItCovers.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = '\t\tc.Color = ink\n'
assert s.count(old) == 1, 'm643'
p.write_text(s.replace(old, '\t\t_ = ink\n', 1))
