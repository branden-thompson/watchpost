import pathlib
# 0.19.0 FR-5.4: an offset more than two spreads from the typical is said.
# Killed by TestAnUnusualOffsetIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tcase o.Apart():\n'
assert old in s, 'm281'
p.write_text(s.replace(old, '\tcase false:\n'))
