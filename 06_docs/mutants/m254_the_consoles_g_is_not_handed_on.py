import pathlib
# 0.19.0 FR-2.1, D-195: the console hands g to the window it opens.
# Killed by TestTheConsoleOpensTheMapOnTheStation.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = 'actQuit, actLookup, actMap:'
assert old in s, 'm254'
p.write_text(s.replace(old, 'actQuit, actLookup:'))
