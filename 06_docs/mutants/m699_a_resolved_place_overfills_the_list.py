import pathlib
# 0.19.5 Q2.7 (RN-8): a place resolved into a full watchlist is refused.
# Killed by TestTheWatchlistStopsAtItsCap.
p = pathlib.Path('modes/tty/modal_location.go'); s = p.read_text()
old = '\t\tif len(watch) >= WatchCap {\n'
assert s.count(old) == 1, 'm699'
p.write_text(s.replace(old, '\t\tif len(watch) > WatchCap {\n', 1))
