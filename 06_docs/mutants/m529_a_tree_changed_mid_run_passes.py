import pathlib
# 0.19.0 RC-3: a tree changed during the run is recorded green.
# Killed by TestATreeChangedDuringTheRunIsNoPass.
p = pathlib.Path('tools/runlog/main.go'); s = p.read_text()
old = '\tcase tree != tested:\n'
assert s.count(old) == 1, 'm529'
p.write_text(s.replace(old, '\tcase false:\n', 1))
