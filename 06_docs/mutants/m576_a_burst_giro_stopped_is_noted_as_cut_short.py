import pathlib
# 0.19.0 D-297, A-94 (1) (R6T-6 w3, R6L-4 Mw): the diagnostics note a burst GIRO itself stopped as cut short by the deadline.
# Killed by TestACutShortBurstIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'n > 0 && n != was.StationsNotReached && snap.Inputs.GIRO == ionomaps.GIROAnswered {'
assert s.count(old) == 1, 'm576'
p.write_text(s.replace(old, 'n > 0 && n != was.StationsNotReached {', 1))
