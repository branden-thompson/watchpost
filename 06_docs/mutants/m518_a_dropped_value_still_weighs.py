import pathlib
# 0.19.0 D-242: a value forgotten or replaced still weighs in the memo's bound.
# Killed by TestAWeightBoundHoldsTheValues.
p = pathlib.Path('platform/agememo/agememo.go'); s = p.read_text()
old = '\t\tm.weight -= e.weight\n\t\tdelete(m.items, k)\n'
assert s.count(old) == 1, 'm518'
p.write_text(s.replace(old, '\t\tm.weight -= e.weight * 0\n\t\tdelete(m.items, k)\n', 1))
