import pathlib
# 0.19.0 W0.2: a host's own cap, tighter than the client's, holds.
# Killed by TestEachSourceHasItsOwnCap.
p = pathlib.Path('platform/httpx/plain.go'); s = p.read_text()
old = '\tif own, ok := p.hostMax[host]; ok && own > 0 && own < limit {'
assert old in s, 'm319'
p.write_text(s.replace(old, '\tif own, ok := p.hostMax[host]; false && ok && own > 0 && own < limit {'))
