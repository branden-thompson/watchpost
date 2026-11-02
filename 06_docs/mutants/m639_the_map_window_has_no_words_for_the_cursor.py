import pathlib
# 0.19.3 A-107 (12): the map window's cursor stands on its words.
# Killed by TestTheMapPlacesTheCursorOnItsWords.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = "\tif d.modal == modalMap {\n"
assert s.count(old) == 1, 'm639'
p.write_text(s.replace(old, "\tif false {\n", 1))
