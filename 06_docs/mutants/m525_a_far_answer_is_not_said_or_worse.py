import pathlib
# 0.19.0 RB-1 (D-129, FR-5.5): beyond the far distance the words drop "or worse".
# Killed by TestEachTargetSaysItsStation.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = ' away, about as good as typical here, or worse"'
assert s.count(old) == 1, 'm525'
p.write_text(s.replace(old, ' away, about as good as typical"', 1))
