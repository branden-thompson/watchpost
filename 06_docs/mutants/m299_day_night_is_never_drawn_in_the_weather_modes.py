import pathlib
# 0.19.0 D-199: Day / night switched on draws the night in the weather modes.
# Killed by TestDayNightIsAMapDetailOption.
p = pathlib.Path('modes/tty/map_night.go'); s = p.read_text()
old = '\tcase !d.detailOn(nightDetailKey):\n'
assert old in s, 'm299'
p.write_text(s.replace(old, '\tcase true:\n'))
