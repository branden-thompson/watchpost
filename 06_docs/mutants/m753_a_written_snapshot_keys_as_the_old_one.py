import pathlib
# 0.19.5 D-31 (PF-9): a snapshot written straight onto the model stands for itself, never for the last one that arrived.
# Killed by TestAWrittenSnapshotStandsForItself.
p = pathlib.Path('modes/tty/memo.go'); s = p.read_text()
old = '\tif s != nil && s == c.seen {\n\t\treturn c.rep\n'
assert s.count(old) == 1, 'm753'
p.write_text(s.replace(old, '\tif s != nil && c.seen != nil {\n\t\treturn c.rep\n', 1))
