import pathlib
# 0.19.0 BUILD red team round 2 (D-227): one reach at a time, the latest wanted asked once the one in flight is answered.
# Killed by TestHourStepsAskOneReachAtATime.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '	if which == answerReach && d.mapPane.propReachAsked != (propReachKey{}) && d.mapPane.propReachAsked != d.mapPane.propReach.key {'
assert old in s, 'm491'
p.write_text(s.replace(old, '	if which == answerReach && d.mapPane.propReachAsked != (propReachKey{}) && false {', 1))
