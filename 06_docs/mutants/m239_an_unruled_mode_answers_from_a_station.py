import pathlib
# 0.19.0 D-193: a UI mode with no role explores from the centre.
# Killed by TestEveryUIModeHasAPropagationRole.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = '\t\treturn propRole{origin: propFromCentre}\n'
assert old in s, 'm239'
p.write_text(s.replace(old, '\t\treturn propRole{origin: propFromStation}\n'))
