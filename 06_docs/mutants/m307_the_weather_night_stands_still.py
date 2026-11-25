import pathlib
# 0.19.0 D-199: in Radar mode the night follows the clock.
# Killed by TestDayNightIsAMapDetailOption.
p = pathlib.Path('modes/tty/map_night.go'); s = p.read_text()
old = '\tif on == (len(mm.mapPane.nightGiven) > 0) && (!on || at.Equal(mm.mapPane.nightAt)) {\n'
assert old in s, 'm307'
p.write_text(s.replace(old, '\tif on == (len(mm.mapPane.nightGiven) > 0) && (!on || true || at.Equal(mm.mapPane.nightAt)) {\n'))
