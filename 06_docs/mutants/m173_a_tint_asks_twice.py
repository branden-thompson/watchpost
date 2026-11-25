import pathlib
# FR-4.2: a weather tint chosen in the Propagation mode asks the temperature once.
# Killed by TestAWeatherTintFromPropagationAsksOnce.
p = pathlib.Path('modes/tty/map_prop.go'); s = p.read_text()
old = '\t\tw = w.and(toggled)\n'
assert old in s, 'm173'
p.write_text(s.replace(old, '\t\tw = w.and(toggled)\n\t\tnd, again := nd.askTemp(e)\n\t\tcmd = tea.Batch(cmd, again)\n'))
