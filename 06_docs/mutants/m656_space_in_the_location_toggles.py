import pathlib
# 0.19.5 D-26 (DEF-6): space in the location is a letter of the place's name.
# Killed by TestTheRequestWindowsKeysDoWhatTheyClaim.
p = pathlib.Path('modes/tty/request.go'); s = p.read_text()
old = '\t\tif d.request.field == requestLocation {\n\t\t\treturn d.requestType(" ")\n'
assert s.count(old) == 1, 'm656'
p.write_text(s.replace(old, '\t\tif false {\n\t\t\treturn d.requestType(" ")\n', 1))
