import pathlib
# 0.19.0 D-109: during a storm the hour ahead's words say typical may not hold.
# Killed by TestAStormSaysTypicalMayNotHold.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '\tif propStorm(d.mapPane.propSnap.Scales) {\n\t\twords ='
assert old in s, 'm361'
p.write_text(s.replace(old, '\tif false && propStorm(d.mapPane.propSnap.Scales) {\n\t\twords ='))
