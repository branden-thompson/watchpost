import pathlib
# 0.19.0 D-193: the Broadcaster answers from its transmitter when one is set.
# Killed by TestTheOriginFollowsTheUIMode.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = 'if t := d.cfg.Transmitter; t != nil && (t.Lat != 0 || t.Lon != 0) {'
assert old in s, 'm240'
p.write_text(s.replace(old, 'if t := d.cfg.Transmitter; false && t != nil && (t.Lat != 0 || t.Lon != 0) {'))
