import pathlib
# 0.19.5 Q2.5 (RN-6): Change Position shows why a typed position was refused.
# Killed by TestChangePositionSaysItsRangeAndItsError.
p = pathlib.Path('modes/tty/broadcaster_manage.go'); s = p.read_text()
old = '\tif d.cardErr != "" {\n\t\trows = append(rows, strings.Repeat'
assert s.count(old) == 1, 'm695'
p.write_text(s.replace(old, '\tif false {\n\t\trows = append(rows, strings.Repeat', 1))
