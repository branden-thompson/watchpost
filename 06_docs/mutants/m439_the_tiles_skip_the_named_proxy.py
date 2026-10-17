import pathlib
# 0.19.0 D-216: the tiles reach the proxy the environment names through go-tuiMaps' CheckedDialerFor.
# Killed by TestTheTilesReachALocalProxy.
p = pathlib.Path('app/maptiles.go'); s = p.read_text()
old = '\treturn &http.Transport{Proxy: proxy, DialContext: dial,'
assert old in s, 'm439'
p.write_text(s.replace(old, '\t_ = proxy\n\treturn &http.Transport{Proxy: nil, DialContext: dial,'))
