import pathlib
# 0.19.0 D-213: the next run takes the throttle state kept.
# Killed by TestTheThrottleStateOutlivesARestart.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\ts.restoreState()\n\treturn s, nil\n'
assert old in s, 'm397'
p.write_text(s.replace(old, '\treturn s, nil\n'))
