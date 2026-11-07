import pathlib
# 0.19.5 Q4.3 (PF-8): setTicker names the new contents, so the tape memo cannot serve the old ones.
# Killed by TestTheTapeKeyCoversEverythingTheTapeShows.
p = pathlib.Path('modes/tty/ticker.go'); s = p.read_text()
old = '\td.ticker, d.tickerGen = items, tickerGens.Add(1)\n'
assert s.count(old) == 1, 'm742'
p.write_text(s.replace(old, '\td.ticker = items\n', 1))
