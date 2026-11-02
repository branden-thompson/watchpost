import pathlib
# 0.19.3 D-366: while the terminal is not focused, the cursor wears its cell's colour.
# Killed by TestTheCursorWearsTheColourItCovers.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = '\tlook := cursorLook{colour: cell.Ink, coloured: true, shape: tea.CursorBlock}\n'
assert s.count(old) == 1, 'm643'
p.write_text(s.replace(old, '\tlook := cursorLook{colour: cell.Ink, shape: tea.CursorBlock}\n', 1))
