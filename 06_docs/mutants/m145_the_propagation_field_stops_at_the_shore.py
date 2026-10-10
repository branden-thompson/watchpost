import pathlib
# 0.19.0 W3.1 (FR-1.6, D-32): the field continues over the sea.
# Killed by TestAPropagationFieldCrossesTheSea.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = 'Values: values, Lines: true, OverWater: true}'
assert old in s, 'm145'
p.write_text(s.replace(old, 'Values: values, Lines: true, OverWater: false}'))
