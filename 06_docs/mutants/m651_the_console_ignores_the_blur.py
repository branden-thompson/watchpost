import pathlib
# 0.19.3 D-367: the console takes the terminal's blur.
# Killed by TestTheCursorWearsTheColourItCovers.
p = pathlib.Path('modes/tty/broadcaster.go'); s = p.read_text()
old = '\t\tb.blurred = true\n'
assert s.count(old) == 1, 'm651'
p.write_text(s.replace(old, '\t\tb.blurred = false\n', 1))
