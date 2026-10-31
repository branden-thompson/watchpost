import pathlib
# 0.19.0 D-199: switching Day / night hands the night in at once.
# Killed by TestDayNightIsAMapDetailOption.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = "\treturn d.setNight() // Day / night is the map's own overlay (D-199)"
assert old in s, 'm301'
p.write_text(s.replace(old, "\treturn d // Day / night is the map's own overlay (D-199)"))
