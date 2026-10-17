import pathlib
# 0.19.0 D-213: the throttle state is kept after each update.
# Killed by TestTheThrottleStateOutlivesARestart.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\t\t\ts.keepState()\n'
assert old in s, 'm396'
p.write_text(s.replace(old, ''))
