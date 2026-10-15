import pathlib
# 0.19.0 D-193: the selected place standing in for a transmitter says so.
# Killed by TestTheOriginFollowsTheUIMode.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = 'name: loc.Label + " " + bcNoTransmitter, ok: true}'
assert old in s, 'm241'
p.write_text(s.replace(old, 'name: loc.Label, ok: true}'))
