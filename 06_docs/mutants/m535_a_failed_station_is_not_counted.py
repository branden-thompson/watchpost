import pathlib
# 0.19.0 D-247: a station whose own ask failed is not counted.
# Killed by TestTheLibrarysRealStatesAreSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'switch n := s.Inputs.StationsFailed; {'
assert s.count(old) == 1, 'm535'
p.write_text(s.replace(old, 'switch n := 0; {', 1))
