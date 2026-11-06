import pathlib
# 0.19.5 Q2.8 (JM-8): Reduce motion's arrow branch is reached before the tone toggle's.
# Killed by TestEveryArrowRowChangesSomething.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\t\tif d.setup.focus == rowReduceMotion {\n\t\t\td = d.cycleReduceMotion('
assert s.count(old) == 1, 'm706'
p.write_text(s.replace(old, '\t\tif false {\n\t\t\td = d.cycleReduceMotion(', 1))
