import pathlib
# 0.19.0 D-199: Day / night is off until switched on, at every level.
# Killed by TestDayNightIsAMapDetailOption.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = 'const detailNever = tuimaps.DetailFull + 1'
assert old in s, 'm300'
p.write_text(s.replace(old, 'const detailNever = tuimaps.DetailEssential'))
