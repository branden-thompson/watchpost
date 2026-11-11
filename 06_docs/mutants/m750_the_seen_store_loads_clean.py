import pathlib
# 0.19.5 Q4.6: the store loads dirty, so the first save of a session writes and
# tightens the file's mode (R3-D-01).
# Killed by TestTheSeenStoreWritesOnlyWhatChanged.
p = pathlib.Path('app/seen_store.go'); s = p.read_text()
old = 'ids: map[string]time.Time{}, dirty: true}'
assert s.count(old) == 1, 'm750'
p.write_text(s.replace(old, 'ids: map[string]time.Time{}}', 1))
