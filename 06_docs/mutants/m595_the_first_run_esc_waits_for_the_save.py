import pathlib
# 0.19.0 D-336: esc on a first run holds the window open until the save lands.
# Killed by TestTheFirstRunSetupsEscKeepsTheLocation.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\t\t\tnext := m.(Dashboard).close()\n'
assert s.count(old) == 1, 'm595'
p.write_text(s.replace(old, '\t\t\tnext := m.(Dashboard)\n', 1))
