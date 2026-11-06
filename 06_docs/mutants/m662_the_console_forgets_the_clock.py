import pathlib
# 0.19.5 D-21: the console's masthead is stamped in the clock Setting.
# Killed by TestTheConsolesMastheadWearsTheListenersClock.
p = pathlib.Path('modes/tty/broadcaster.go'); s = p.read_text()
old = 'Units: b.units, Clock: b.clockFmt, ASCII: b.ascii'
assert s.count(old) == 1, 'm662'
p.write_text(s.replace(old, 'Units: b.units, ASCII: b.ascii', 1))
