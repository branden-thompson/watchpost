import pathlib
# D-158: a weather tint chosen in the Propagation mode is the one drawn on leaving it.
# Killed by TestAWeatherTintLeavesThePropagationMode.
p = pathlib.Path('modes/tty/map_prop.go'); s = p.read_text()
old = '\t\tnd := d.chooseTint(r.key)'
assert old in s, 'm155'
p.write_text(s.replace(old, '\t\tnd := d'))
