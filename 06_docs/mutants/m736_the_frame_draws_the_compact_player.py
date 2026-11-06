import pathlib
# 0.19.5 Q3.3 (PF-3): the frame renders the player of the mode in force, and only that one.
# Killed by TestTheRadiosPlannedHeightIsItsRows.
p = pathlib.Path('modes/tty/layout.go'); s = p.read_text()
old = '\tfl.radioRows = d.radioRender(drawn, plan, fl.compact)\n'
assert s.count(old) == 1, 'm736'
p.write_text(s.replace(old, '\tfl.radioRows = d.radioRender(drawn, plan, true)\n', 1))
