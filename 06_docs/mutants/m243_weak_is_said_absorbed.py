import pathlib
# 0.19.0 D-70, D-193: under a LUF the circuit decides, too weak, not absorbed.
# Killed by TestTheTargetSaysWhatIsUsable.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = 'l.weak || b.Limit == ionomaps.LimitCircuit'
assert old in s, 'm243'
p.write_text(s.replace(old, 'false'))
