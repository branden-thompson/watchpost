import pathlib
# 0.19.0 D-199: in Radar mode the night moves with the clock.
# Killed by TestDayNightIsAMapDetailOption.
p = pathlib.Path('modes/tty/map_night.go'); s = p.read_text()
old = '\tif d.modal != modalMap || d.mapPane.m == nil {\n'
assert old in s, 'm199'
p.write_text(s.replace(old, '\tif true {\n'))
