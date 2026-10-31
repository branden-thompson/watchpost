import pathlib
# 0.19.0 D-195: the console's Help lists the map's own keys.
# Killed by TestTheBroadcastersMapOffersNoPlaceKeys.
p = pathlib.Path('modes/tty/help_about.go'); s = p.read_text()
old = "\t\t}, windowHelpGroups()...) // the map's own keys, as its role binds them (D-195)\n"
assert old in s, 'm267'
p.write_text(s.replace(old, '\t\t}, []helpGroup{}...)\n'))
