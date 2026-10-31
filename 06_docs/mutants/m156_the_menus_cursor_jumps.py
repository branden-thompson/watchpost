import pathlib
# D-158: the cursor stays on the Radio Propagation row as the mode changes.
# Killed by TestTheOverlaysMenuOffersThePropagationMode.
p = pathlib.Path('modes/tty/map_prop.go'); s = p.read_text()
old = '\t\tnd.mapPane.menuAt = at // the tints lead the menu in both modes: the cursor stays on its row'
assert old in s, 'm156'
p.write_text(s.replace(old, '\t\tnd.mapPane.menuAt = at + 1 // the tints lead the menu in both modes: the cursor stays on its row'))
