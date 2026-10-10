import pathlib
# D-161, D-162: Radio Propagation is the OVERLAYS group's fourth row; it opens no MAP DETAIL section.
# Killed by TestThePropagationRowIsTheFourthOverlay.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = 'label: propagationRowLabel, weather: true})'
assert old in s, 'm158'
p.write_text(s.replace(old, 'label: propagationRowLabel, weather: false})'))
