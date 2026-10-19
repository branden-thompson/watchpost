import pathlib
# 0.19.0 D-239: a file not read is said replaced by the last held, though none may be held.
# Killed by TestEverySnapshotsStateIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '+" not read this update")'
assert s.count(old) == 1, 'm512'
p.write_text(s.replace(old, '+" not read: the last held used")', 1))
