import pathlib
# D-164: the place's name in the bold bright tone.
# Killed by TestRoundFourMarksStandOut.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tplace := tint(name, render.MapPropPlace) + "   "'
assert old in s, 'm161'
p.write_text(s.replace(old, '\tplace := name + "   "'))
