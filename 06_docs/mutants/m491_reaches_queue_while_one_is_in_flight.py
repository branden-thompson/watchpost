import pathlib
# 0.19.0 BUILD red team round 2 (D-227): one reach at a time, the latest wanted asked once the one in flight is answered.
# Killed by TestHourStepsAskOneReachAtATime.
p = pathlib.Path('modes/tty/map_prop_freq.go'); s = p.read_text()
old = '\tif d.mapPane.propReachAsked != (propReachKey{}) && d.mapPane.propReachAsked != d.mapPane.propReach.key {'
assert s.count(old) == 1, 'm491'
p.write_text(s.replace(old, '\tif d.mapPane.propReachAsked != (propReachKey{}) && false {', 1))
