import pathlib
# 0.19.0 D-336: esc on the first-run Setup drops the location chosen.
# Killed by TestTheFirstRunSetupsEscKeepsTheLocation.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\t\tif d.cfg.FirstRun && d.setup.ref != nil {\n'
assert s.count(old) == 1, 'm592'
p.write_text(s.replace(old, '\t\tif false && d.cfg.FirstRun && d.setup.ref != nil {\n', 1))
