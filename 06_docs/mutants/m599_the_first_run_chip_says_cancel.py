import pathlib
# 0.19.0 VALIDATE VR-2: the first run's esc saves the pick, and its chip still says Cancel.
# Killed by TestTheFirstRunSetupsEscKeepsTheLocation.
p = pathlib.Path('modes/tty/setup_layout.go'); s = p.read_text()
old = '\t\tcloseLabel = "Save" // the first run\'s esc keeps the location chosen (D-336)\n'
assert s.count(old) == 1, 'm599'
p.write_text(s.replace(old, '\t\tcloseLabel = "Cancel"\n', 1))
