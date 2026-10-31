import pathlib
# 0.19.0 D-178: F opens the frequency entry over the map.
# Killed by TestYourFrequencyDrawsItsReach.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\treturn d.flashMapKey(act).openPropFreq(), nil, true\n'
assert old in s, 'm206'
p.write_text(s.replace(old, '\t\treturn d.flashMapKey(act), nil, true\n'))
