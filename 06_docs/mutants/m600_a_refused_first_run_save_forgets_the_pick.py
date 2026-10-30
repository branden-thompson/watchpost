import pathlib
# 0.19.0 VALIDATE VC-1: a first-run save refused (a mistyped key) reopens the window with the pick gone.
# Killed by TestAFirstRunSaveRefusedKeepsThePick.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\t\t\treturn m.(Dashboard).close(), save\n'
assert s.count(old) == 1, 'm600'
p.write_text(s.replace(old, '\t\t\tnext := m.(Dashboard).close()\n\t\t\tnext.setup = setupState{}\n\t\t\treturn next, save\n', 1))
