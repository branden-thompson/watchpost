import pathlib
# 0.19.0 BUILD red team round 2 (D-228): the lead's interval is taken on the same draw as the agreements.
# Killed by TestTheIntervalKeepsThePairing.
p = pathlib.Path('tools/reachscore/interval.go'); s = p.read_text()
old = '100*float64(hits[shipped]-hits[climatology])/float64(n)'
assert old in s, 'm490'
p.write_text(s.replace(old, '100*float64(hits[shipped]-hits[background])/float64(n)', 1))
