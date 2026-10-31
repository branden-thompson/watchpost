import pathlib
# 0.19.0 FR-2.1, D-195: g opens the map on the console.
# Killed by TestTheConsoleOpensTheMapOnTheStation.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = '\t\tactMap:           {Keys: []string{"g"}, Help: "Map"}, // Observer\'s key and window, on the station\'s place (D-56, D-195)\n'
assert old in s, 'm253'
p.write_text(s.replace(old, ''))
