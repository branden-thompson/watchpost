import pathlib
# D-169: the propagation client's asks are in the counters the Status window reads.
# Killed by TestThePropagationClientsAsksReachTheStatusWindow.
p = pathlib.Path('app/stats.go'); s = p.read_text()
old = 'lp.tiles.stats(), lp.propagation.requestStats())'
assert old in s, 'm183'
p.write_text(s.replace(old, 'lp.tiles.stats())'))
