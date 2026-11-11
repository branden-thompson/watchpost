import pathlib
# 0.19.5 Q9.7 (RR-12): the picker lights the arrow that was not pressed.
# Killed by TestEveryArrowRowsOutcomeIsPinned.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\tif forward {\n\t\td.setup.flash = flashRight\n\t}\n\treturn d.settled()\n'
assert s.count(old) == 1, 'm800'
p.write_text(s.replace(old, '\tif !forward {\n\t\td.setup.flash = flashRight\n\t}\n\treturn d.settled()\n', 1))
