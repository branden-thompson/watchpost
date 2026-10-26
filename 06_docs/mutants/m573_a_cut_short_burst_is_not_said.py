import pathlib
# 0.19.0 D-297: a burst the update's deadline cut short is said as a whole burst.
# Killed by TestACutShortBurstIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif n := s.Inputs.StationsNotReached; n > 0 && s.Inputs.GIRO == ionomaps.GIROAnswered {\n'
assert s.count(old) == 1, 'm573'
p.write_text(s.replace(old, '\tif n := s.Inputs.StationsNotReached; n > 1<<30 && s.Inputs.GIRO == ionomaps.GIROAnswered {\n', 1))
