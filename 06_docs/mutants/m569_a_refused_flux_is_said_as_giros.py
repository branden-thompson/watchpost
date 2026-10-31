import pathlib
# 0.19.0 R3L-6, R3C-2: a refused solar flux is said as GIRO's throttle state starting as new.
# Killed by TestTheThrottleStateOutlivesARestart.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\tif giro {\n'
assert s.count(old) == 1, 'm569'
p.write_text(s.replace(old, '\tif giro || mean {\n', 1))
