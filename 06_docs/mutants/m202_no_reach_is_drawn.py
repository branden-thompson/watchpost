import pathlib
# 0.19.0 FR-1.16 (D-178): the reach is handed to the map over the field.
# Killed by TestYourFrequencyDrawsItsReach.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\t\tout = append(out, propReachOverlay(reach, h.At)) // after the field: its edge alone over it (L-2.6)\n'
assert old in s, 'm202'
p.write_text(s.replace(old, '\t\t_ = reach\n'))
