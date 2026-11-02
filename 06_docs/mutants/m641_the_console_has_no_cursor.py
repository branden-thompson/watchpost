import pathlib
# 0.19.3 D-362: the console's cursor is on its focused row.
# Killed by TestTheConsolePlacesTheCursor.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = '\t\tx, y, ok := r.broadcaster.consoleCursor(v.Content)\n'
assert s.count(old) == 1, 'm641'
p.write_text(s.replace(old, '\t\tx, y, ok := r.broadcaster.consoleCursor(v.Content)\n\t\tok = false\n', 1))
