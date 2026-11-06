import pathlib
# 0.19.5 Q2.11 (RN-8): the full watchlist's note names the cap itself, WatchCap.
# Killed by TestTheWatchlistStopsAtItsCap.
p = pathlib.Path('modes/tty/modal_location.go'); s = p.read_text()
old = 'fmt.Sprintf("  Only %d locations are allowed in the priority list", WatchCap)'
assert s.count(old) == 1, 'm720'
p.write_text(s.replace(old, 'fmt.Sprintf("  Only %d locations are allowed in the priority list", WatchCap+1)', 1))
