import pathlib
# D-166: a tick is armed while the age is shown, so the frame moves.
# Killed by TestThePropagationAgeMovesWithTheClock.
p = pathlib.Path('modes/tty/dashboard.go'); s = p.read_text()
old = '\tcase d.propAgeShown(): // the Propagation snapshot'
assert old in s, 'm172'
p.write_text(s.replace(old, '\tcase false: // the Propagation snapshot'))
