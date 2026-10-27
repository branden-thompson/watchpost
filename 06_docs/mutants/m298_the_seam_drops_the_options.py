import pathlib
# 0.19.0 FR-4.8, D-187, D-105: the window's options reach go-ionomaps whole.
# Killed by TestTheWindowsOptionsReachTheLibrary.
p = pathlib.Path('app/mapprop.go'); s = p.read_text()
old = '\t\tr := <-svc.Ask(ask, opts)'
assert old in s, 'm298'
p.write_text(s.replace(old, '\t\tr := <-svc.Ask(ask, ionomaps.UpdateOptions{})'))
