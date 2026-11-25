import pathlib
# 0.19.0 D-199: Day / night is no level's, so it leaves the level's name.
# Killed by TestDayNightIsAMapDetailOption.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = 'if !l.overlay && mm.detailOn(l.key) != (l.level <= mm.mapDetailLevel) {'
assert old in s, 'm303'
p.write_text(s.replace(old, 'if mm.detailOn(l.key) != (l.level <= mm.mapDetailLevel) {'))
