import pathlib
# 0.19.0 D-199: the Propagation mode always draws the night, so offers no switch.
# Killed by TestThePropagationModeAlwaysDrawsTheNight.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = '\t\treturn append(out, detailRows(nightDetailKey)...)'
assert old in s, 'm305'
p.write_text(s.replace(old, '\t\treturn append(out, detailRows("")...)'))
