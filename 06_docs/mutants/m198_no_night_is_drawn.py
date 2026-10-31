import pathlib
# 0.19.0 D-174: the Propagation mode draws the night.
# Killed by TestTheNightIsDrawnOnThePropagationMap.
p = pathlib.Path('modes/tty/map_night.go'); s = p.read_text()
old = '\t\treturn d.propNightMoment(), d.mapPane.propSnap != nil\n'
assert old in s, 'm198'
p.write_text(s.replace(old, '\t\treturn d.propNightMoment(), false\n'))
