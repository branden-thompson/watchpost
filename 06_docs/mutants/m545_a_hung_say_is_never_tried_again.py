import pathlib
# 0.19.0 D-276: a render killed at its bound fails at once, never tried once more.
# Killed by TestAHungRenderIsTriedOnceMore.
p = pathlib.Path('domains/radio/synth/synth_test.go'); s = p.read_text()
old = '\t\tif !hung || *retried {\n'
assert s.count(old) == 1, 'm545'
p.write_text(s.replace(old, '\t\tif true || !hung || *retried {\n', 1))
