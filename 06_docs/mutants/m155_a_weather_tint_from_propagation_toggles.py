import pathlib
# D-158: a weather tint chosen in the Propagation mode is drawn, never cleared.
# Killed by TestAWeatherTintLeavesThePropagationMode.
p = pathlib.Path('modes/tty/map_prop.go'); s = p.read_text()
old = '\t\tnd = nd.chooseTint(r.key)'
assert old in s, 'm155'
p.write_text(s.replace(old, '\t\tnd = nd.switchRow(r)'))
