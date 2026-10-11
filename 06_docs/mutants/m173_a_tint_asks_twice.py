import pathlib
# FR-4.2: a weather tint chosen in the Propagation mode asks the temperature once.
# Killed by TestAWeatherTintFromPropagationAsksOnce.
p = pathlib.Path('modes/tty/map_prop.go'); s = p.read_text()
old = '\t\tsave := nd.uiApplyCmd()\n'
assert old in s, 'm173'
p.write_text(s.replace(old, '\t\tnd, again := nd.askTemp()\n\t\tsave := tea.Batch(nd.uiApplyCmd(), again)\n'))
