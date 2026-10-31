import pathlib
# 0.19.0 D-324 (P10-03): the shares grow with every host named.
# Killed by TestTheSharesAreBounded.
p = pathlib.Path('platform/httpx/httpx.go'); s = p.read_text()
old = '\tif len(c.shares) >= maxShareHosts {\n'
assert s.count(old) == 1, 'm587'
p.write_text(s.replace(old, '\tif len(c.shares) >= maxShareHosts*100 {\n', 1))
