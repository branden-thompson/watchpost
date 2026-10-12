import pathlib
# 0.19.0 D-134: an hour ahead gives its typical error at the place.
# Killed by TestTheTypicalErrorStatesItsBasis.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '\t\tlabel += fmt.Sprintf(" · ±%.1f MHz", e)\n'
assert old in s, 'm211'
p.write_text(s.replace(old, '\t\t_ = e\n'))
