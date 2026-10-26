import pathlib
# 0.19.0 D-297: a burst's count of stations not reached is noted for every snapshot of the hour, flooding the diagnostics.
# Killed by TestACutShortBurstIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'n > 0 && n != was.StationsNotReached && snap.Inputs.GIRO == ionomaps.GIROAnswered {'
assert s.count(old) == 1, 'm574'
p.write_text(s.replace(old, 'n > 0 && snap.Inputs.GIRO == ionomaps.GIROAnswered {', 1))
