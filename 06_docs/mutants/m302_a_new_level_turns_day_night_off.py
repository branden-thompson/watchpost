import pathlib
# 0.19.0 D-199: a new detail level leaves Day / night as it was.
# Killed by TestDayNightIsAMapDetailOption.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = '\tif night {\n'
assert old in s, 'm302'
p.write_text(s.replace(old, '\tif night && false {\n'))
