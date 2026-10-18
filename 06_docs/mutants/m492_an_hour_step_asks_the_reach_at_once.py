import pathlib
# 0.19.0 BUILD red team round 2 (D-227): hour steps settle as pans do before the reach is asked.
# Killed by TestHourStepsAskOneReachAtATime.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = 'if moving && (which == answerCentre || which == answerReach || d.propRole().origin == propFromCentre) {'
assert old in s, 'm492'
p.write_text(s.replace(old, 'if moving && (which == answerCentre || d.propRole().origin == propFromCentre) {', 1))
