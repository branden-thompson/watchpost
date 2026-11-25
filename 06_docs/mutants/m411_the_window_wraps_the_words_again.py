import pathlib
# 0.19.0 D-211: text that will scroll is wrapped clear of the rail.
# Killed by TestARefreshKeepsTheReadersPlace.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\tif len(text) > e.rows() {\n'
assert old in s, 'm411'
p.write_text(s.replace(old, '\t\tif len(text) > e.rows() && false {\n'))
