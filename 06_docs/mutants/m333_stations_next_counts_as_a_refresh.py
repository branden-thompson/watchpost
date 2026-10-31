import pathlib
# 0.19.0 FR-3.6: GIRO's stations following NOAA's field at once complete the first answer.
# Killed by TestARefreshIsSaidOnce.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\tif snap.Reason != ionomaps.StationsNext {\n'
assert old in s, 'm333'
p.write_text(s.replace(old, '\t\tif true {\n'))
