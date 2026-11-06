import pathlib
# 0.19.5 Q2.8 (JM-8): the Maps row's arrow branch is reached before the tone toggle's.
# Killed by TestEveryArrowRowChangesSomething.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\t\tif d.setup.focus == rowMapsOn {\n'
assert s.count(old) == 1, 'm707'
p.write_text(s.replace(old, '\t\tif false {\n', 1))
