import pathlib
# 0.19.0 D-27: a found place is not kept past the mode.
# Killed by TestAFoundPlaceIsAnsweredAndNotKept.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\treturn d.markFound(nil) // a found place is not kept past the mode (D-27)\n'
assert old in s, 'm388'
p.write_text(s.replace(old, '\treturn d\n'))
