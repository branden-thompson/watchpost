import pathlib
# 0.19.3 A-107 (12): while the Description is open the cursor is on its first line.
# Killed by TestTheMapPlacesTheCursorOnItsWords.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = "\t\tif d.mapPane.alertsOn && i+1 < len(lines)"
assert s.count(old) == 1, 'm640'
p.write_text(s.replace(old, "\t\tif false && d.mapPane.alertsOn && i+1 < len(lines)", 1))
