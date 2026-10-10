import pathlib
# D-163: the block's blank row is counted in the rows under the map, so the window's body is its height.
# Killed by TestThePropagationRowsFollowTheMock.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = "\t\trows += propBlockRows - (radarRows + 2) // its block's blank row before the place (D-163)"
assert old in s, 'm159'
p.write_text(s.replace(old, "\t\trows += 0 // its block's blank row before the place (D-163)"))
