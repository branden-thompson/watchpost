import pathlib
# 0.19.5 Q4.2 (PF-7, D-22): the masthead memo keys on the stamp's age words, which change with the clock and nothing else.
# Killed by TestTheMastheadKeyCoversEverythingTheMastheadShows.
p = pathlib.Path('modes/tty/masthead.go'); s = p.read_text()
old = 'k.hasData, k.asOf, k.ago, k.stale = true, at.UnixNano(), agoWords(now.Sub(at)), now.Sub(at) > staleAfter'
assert s.count(old) == 1, 'm744'
p.write_text(s.replace(old, 'k.hasData, k.asOf, k.stale = true, at.UnixNano(), now.Sub(at) > staleAfter', 1))
