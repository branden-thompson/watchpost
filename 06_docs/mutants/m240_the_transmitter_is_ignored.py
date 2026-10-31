import pathlib
# 0.19.0 D-193: the Broadcaster answers from its transmitter when one is set.
# Killed by TestTheOriginFollowsTheUIMode.
p = pathlib.Path('modes/tty/map_role.go'); s = p.read_text()
old = '\treturn t == nil || (t.Lat == 0 && t.Lon == 0)\n'
assert old in s, 'm240'
p.write_text(s.replace(old, '\t_ = t\n\treturn true\n'))
