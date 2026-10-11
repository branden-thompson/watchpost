import pathlib
# D-166: the snapshot's age moves with the clock in the window's frame.
# Killed by TestThePropagationAgeMovesWithTheClock.
p = pathlib.Path('modes/tty/memo.go'); s = p.read_text()
old = ', k.mapPropAge = d.propStatusWords(), d.mapPane.propLayer, d.propAgeMinutes()'
assert old in s, 'm171'
p.write_text(s.replace(old, ' = d.propStatusWords(), d.mapPane.propLayer'))
