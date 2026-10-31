import pathlib
# 0.19.0 FR-8.1 (#12): every position field says why a position is enough.
# Killed by TestEveryPositionFieldHasAReason.
p = pathlib.Path('modes/tty/memo.go'); s = p.read_text()
old = '\tsevereRow                 int // position: a row of the severe index'
assert old in s, 'm379'
p.write_text(s.replace(old, '\tsevereRow                 int // a row of the severe index'))
