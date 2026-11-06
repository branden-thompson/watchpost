import pathlib
# 0.19.5 Q2.7 (RN-8): the watchlist is full at WatchCap places.
# Killed by TestTheWatchlistStopsAtItsCap.
p = pathlib.Path('modes/tty/modal_location.go'); s = p.read_text()
old = 'func (d Dashboard) watchlistFull() bool { return d.numPriority() >= WatchCap }'
assert s.count(old) == 1, 'm698'
p.write_text(s.replace(old, 'func (d Dashboard) watchlistFull() bool { return d.numPriority() > WatchCap }', 1))
