import pathlib
# 0.19.0 D-213, D-124: a state that cannot be taken is noted.
# Killed by TestTheThrottleStateOutlivesARestart.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\ts.note("Propagation: " + restoreWords(err) + " - " + fileOnly(err))\n'
assert old in s, 'm398'
p.write_text(s.replace(old, '\t_ = err\n'))
