import pathlib
# 0.19.0 W2.4, D-156: U asks for an update now.
# Killed by TestURefreshesNowUnderTheThrottle.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\treturn d.askPropagation()\n}'
assert old in s, 'm147'
p.write_text(s.replace(old, '\treturn d, nil\n}'))
