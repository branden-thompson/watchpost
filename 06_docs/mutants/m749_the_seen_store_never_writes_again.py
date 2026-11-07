import pathlib
# 0.19.5 Q4.6: a newly seen id makes the store dirty, so the next save writes it.
# Killed by TestTheSeenStoreWritesOnlyWhatChanged.
p = pathlib.Path('app/seen_store.go'); s = p.read_text()
old = '\tif added || len(s.ids) != before {\n'
assert s.count(old) == 1, 'm749'
p.write_text(s.replace(old, '\tif false && (added || len(s.ids) != before) {\n', 1))
