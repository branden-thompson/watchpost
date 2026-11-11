import pathlib
# 0.19.5 Q9.7 (RR-12): the Theme row reaches the voice picker's cycle, which
# has no list for it, so its arrows change nothing.
# Killed by TestEveryArrowRowsOutcomeIsPinned.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\tcase d.setup.focus == rowTheme:\n'
assert s.count(old) == 1, 'm801'
p.write_text(s.replace(old, '\tcase false:\n', 1))
