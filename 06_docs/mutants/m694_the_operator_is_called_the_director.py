import pathlib
# 0.19.5 Q2.5 (RN-6): a card the operator asked for says Station Operator.
# Killed by TestACardSaysWhoAskedForIt.
p = pathlib.Path('modes/tty/broadcaster_lineup.go'); s = p.read_text()
old = '\t\treturn "Station Operator"\n'
assert s.count(old) == 1, 'm694'
p.write_text(s.replace(old, '\t\treturn "Director"\n', 1))
