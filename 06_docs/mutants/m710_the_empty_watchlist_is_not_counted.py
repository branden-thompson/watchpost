import pathlib
# 0.19.5 D-28, D-30 (DEF-8): the empty state's rows count in the frame's height.
# Killed by TestTheFrameNeverExceedsTheTerminal.
p = pathlib.Path('modes/tty/layout.go'); s = p.read_text()
old = '\tif n := d.numPriority(); n > 0 {\n\t\treturn n\n\t}\n'
assert s.count(old) == 1, 'm710'
p.write_text(s.replace(old, '\tif n := d.numPriority(); n >= 0 {\n\t\treturn n\n\t}\n', 1))
