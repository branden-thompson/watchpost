import pathlib
# 0.19.5 Q2.13 (RN minor): the console's up arrow moves its pointer up.
# Killed by TestTheConsolesArrowsMoveItsPointer.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = '\tif a == actQueuePrev {\n\t\treturn -1\n'
assert s.count(old) == 1, 'm721'
p.write_text(s.replace(old, '\tif a == actQueueNext {\n\t\treturn -1\n', 1))
