import pathlib
# D-163: the place's answer lines stand under NOW.
# Killed by TestThePropagationRowsFollowTheMock.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tunder := strings.Repeat(" ", render.Width(place))'
assert old in s, 'm160'
p.write_text(s.replace(old, '\tunder := "  "'))
