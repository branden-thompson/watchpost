import pathlib
# 0.19.0 R-2.6: a row's words keep its upper limit and add D-RAP's.
# Killed by TestADisturbedBandIsDrawnAndNamed.
p = pathlib.Path('platform/propanswers/limits.go'); s = p.read_text()
old = '\t\t\tl.DRAP = float64(b.LimitMHz)\n'
assert old in s, 'm222'
p.write_text(s.replace(old, '\t\t\tl.DRAP = 0\n'))
