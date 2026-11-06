import pathlib
# 0.19.5 Q2.3 (RN-3): the seen store's cap keeps the newest ids.
# Killed by TestTheSeenStoreKeepsTheNewestIDsAtItsCap.
p = pathlib.Path('app/seen_store.go'); s = p.read_text()
old = '\tsort.Slice(all, func(i, j int) bool { return all[i].at.After(all[j].at) })\n\ts.ids = make('
assert s.count(old) == 1, 'm686'
p.write_text(s.replace(old, '\tsort.Slice(all, func(i, j int) bool { return all[i].at.Before(all[j].at) })\n\ts.ids = make(', 1))
