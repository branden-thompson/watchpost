import pathlib
# 0.19.0 D-192: a row's words add the LUF.
# Killed by TestAClosedBandIsDrawnByItsKind.
p = pathlib.Path('platform/propanswers/limits.go'); s = p.read_text()
old = 'l.LUF, l.Weak = float64(b.LimitMHz), '
assert old in s, 'm228'
p.write_text(s.replace(old, 'l.LUF, l.Weak = 0, '))
