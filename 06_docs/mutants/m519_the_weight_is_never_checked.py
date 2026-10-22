import pathlib
# 0.19.0 D-242: a memo made with NewSized holds past MaxBytes.
# Killed by TestAWeightBoundHoldsTheValues.
p = pathlib.Path('platform/agememo/agememo.go'); s = p.read_text()
old = 'heavy := m.o.MaxBytes > 0 &&'
assert s.count(old) == 1, 'm519'
p.write_text(s.replace(old, 'heavy := m.o.MaxBytes < 0 &&', 1))
