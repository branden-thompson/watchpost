import pathlib
# 0.19.3 D-362: an Observer window over the console takes the cursor.
# Killed by TestTheConsolePlacesTheCursor.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = '\t\t\tv.Content, x, y, ok = r.observer.overlay(v.Content, r.broadcaster.width)\n'
assert s.count(old) == 1, 'm642'
p.write_text(s.replace(old, '\t\t\tv.Content, _, _, _ = r.observer.overlay(v.Content, r.broadcaster.width)\n', 1))
