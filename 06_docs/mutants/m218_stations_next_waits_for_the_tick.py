import pathlib
# 0.19.0 D-183: a snapshot said StationsNext asks the next update at once, not at the tick.
# Killed by TestStationsNextAsksAgainAtOnce.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif v.res.Err == nil && v.res.Snapshot.Early == ionomaps.StationsNext {\n'
assert old in s, 'm218'
p.write_text(s.replace(old, '\tif v.res.Err == nil && v.res.Snapshot.Early == ionomaps.StationsNext && false {\n'))
