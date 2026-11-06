import pathlib
# 0.19.5 D-28, D-30 (DEF-8): on a short terminal the priority table gives way.
# Killed by TestTheFrameNeverExceedsTheTerminal.
p = pathlib.Path('modes/tty/layout.go'); s = p.read_text()
old = '\tif over <= 0 || d.numPriority() == 0 || fl.priorityRows <= minPriorityRows {\n'
assert s.count(old) == 1, 'm708'
p.write_text(s.replace(old, '\tif true || over <= 0 || d.numPriority() == 0 || fl.priorityRows <= minPriorityRows {\n', 1))
