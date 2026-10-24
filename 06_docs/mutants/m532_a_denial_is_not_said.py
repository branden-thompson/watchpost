import pathlib
# 0.19.0 D-247: GIRO refusing the host is not said.
# Killed by TestTheLibrarysRealStatesAreSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tcase ionomaps.GIRODenied:\n\t\treturn "GIRO refused this host: not asked again for a day"\n'
assert s.count(old) == 1, 'm532'
p.write_text(s.replace(old, '\tcase ionomaps.GIRODenied:\n\t\treturn ""\n', 1))
