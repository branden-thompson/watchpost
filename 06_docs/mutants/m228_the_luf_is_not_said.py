import pathlib
# 0.19.0 D-192: a row's words add the LUF.
# Killed by TestAClosedBandIsDrawnByItsKind.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = 'l.luf, l.weak = float64(b.LimitMHz), '
assert old in s, 'm228'
p.write_text(s.replace(old, 'l.luf, l.weak = 0, '))
