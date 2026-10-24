import pathlib
# 0.19.0 D-249: the kept F10.7 mean is used unsaid when the solar file is not read.
# Killed by TestTheLibrarysRealStatesAreSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'if slices.Contains(s.Inputs.Unread, "solar") && s.Inputs.F107Age > 0 {'
assert s.count(old) == 1, 'm534'
p.write_text(s.replace(old, 'if slices.Contains(s.Inputs.Unread, "solar") && s.Inputs.F107Age < 0 {', 1))
