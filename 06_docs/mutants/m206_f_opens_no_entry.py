import pathlib
# 0.19.0 D-178: F opens the frequency entry over the map.
# Killed by TestYourFrequencyDrawsItsReach.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\tnd, w := mm.flashMapKey(act, e).openPropFreq(e)\n\t\treturn nd, nil, w, true\n'
assert old in s, 'm206'
p.write_text(s.replace(old, '\t\tnd, w := mm.flashMapKey(act, e), mapWant{}\n\t\treturn nd, nil, w, true\n'))
