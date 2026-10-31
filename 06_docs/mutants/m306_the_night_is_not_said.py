import pathlib
# 0.19.0 D-199, FR-3.1: the words say the night where it is drawn.
# Killed by TestDayNightIsAMapDetailOption.
p = pathlib.Path('modes/tty/map_night.go'); s = p.read_text()
old = '\tif len(d.mapPane.nightGiven) == 0 {\n\t\treturn ""'
assert old in s, 'm306'
p.write_text(s.replace(old, '\tif true {\n\t\treturn ""'))
