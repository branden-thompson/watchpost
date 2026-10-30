import pathlib
# 0.19.0 D-336: Setup's save leaves the window believing it is still the first run.
# Killed by TestTheFirstRunSetupsEscKeepsTheLocation.
p = pathlib.Path('modes/tty/dashboard.go'); s = p.read_text()
old = '\t\t\td.cfg.FirstRun = false // a file now exists: the first run is over (D-336)\n'
assert s.count(old) == 1, 'm593'
p.write_text(s.replace(old, '', 1))
