import pathlib
# 0.19.0 D-336: Setup's save landing closes whatever window is open, not Setup alone.
# Killed by TestTheFirstRunSetupsEscKeepsTheLocation.
p = pathlib.Path('modes/tty/dashboard.go'); s = p.read_text()
old = '\t\tif d.modal == modalSetup { // the Setup window alone: one the listener opened since stays open (D-336)\n'
assert s.count(old) == 1, 'm594'
p.write_text(s.replace(old, '\t\tif true {\n', 1))
