import pathlib
# 0.19.0 D-242: the history store's caches are not gauged, so no soak sees them.
# Killed by TestTheHistoryStoresCachesAreGauged.
p = pathlib.Path('app/stats.go'); s = p.read_text()
old = '\t\tout = append(out, Gauge{Name: "history.days", Len: days, Bytes: bytes}, Gauge{Name: "history.index", Len: indexed})\n'
assert s.count(old) == 1, 'm520'
p.write_text(s.replace(old, '\t\t_, _, _ = days, bytes, indexed\n', 1))
