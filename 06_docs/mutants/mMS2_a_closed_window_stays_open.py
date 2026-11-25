import pathlib
# 0.19.6 B4.3 (design MS-1's S2 to S4): the Dashboard does not close the window
# the map asked it to close.
# Killed by TestOnlyEnterOrEscCloseIt and TestFocusReturnsAfterAFind.
p = pathlib.Path('modes/tty/observer_map.go'); s = p.read_text()
old = '\tif w.Close {\n\t\td = d.close()\n\t}\n'
assert s.count(old) == 1, 'mMS2'
p.write_text(s.replace(old, '\tif w.Close {\n\t\t_ = d\n\t}\n', 1))
