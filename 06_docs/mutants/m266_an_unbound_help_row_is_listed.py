import pathlib
# 0.19.0 D-195: a map Help row whose keys are unbound is not listed.
# Killed by TestTheBroadcastersMapOffersNoPlaceKeys.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\treturn slices.DeleteFunc(rows, func(r mapHelpRow) bool { return r.keys == "" && r.help == "" })'
assert old in s, 'm266'
p.write_text(s.replace(old, '\treturn rows'))
