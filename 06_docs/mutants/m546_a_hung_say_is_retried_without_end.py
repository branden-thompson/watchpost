import pathlib
# 0.19.0 D-276: a second hang is tried again rather than failing the gate.
# Killed by TestAHungRenderIsTriedOnceMore.
p = pathlib.Path('domains/radio/synth/synth_test.go'); s = p.read_text()
old = '\t\t*retried = true\n'
assert s.count(old) == 1, 'm546'
p.write_text(s.replace(old, '\t\t_ = retried\n', 1))
