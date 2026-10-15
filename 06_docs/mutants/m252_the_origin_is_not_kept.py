import pathlib
# 0.19.0 D-193: the origin is worked out in Update, for the frame to read.
# Killed by TestTheOriginFollowsTheUIMode.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = 'd.mapPane.propOrigin = d.propOriginNow()'
assert old in s, 'm252'
p.write_text(s.replace(old, '_ = d.propOriginNow()'))
