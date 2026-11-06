import pathlib
# 0.19.5 D-28, D-30 (DEF-8): a scrolled priority table keeps the focused favourite in view.
# Killed by TestTheFrameNeverExceedsTheTerminal.
p = pathlib.Path('modes/tty/body.go'); s = p.read_text()
old = '\treturn max(0, d.selected-shown+1)\n'
assert s.count(old) == 1, 'm709'
p.write_text(s.replace(old, '\treturn 0\n', 1))
