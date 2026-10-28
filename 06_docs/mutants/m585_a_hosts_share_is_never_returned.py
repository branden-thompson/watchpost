import pathlib
# 0.19.0 D-324: an answered request keeps its host's share.
# Killed by TestAHostsShareComesBackWithEachAnswer; in the whole package an earlier test asking one
# host past its share never returns, as the app would hang, and the run ends on its timeout.
p = pathlib.Path('platform/httpx/httpx.go'); s = p.read_text()
old = '\t\treturn func() { <-slot; <-share }, nil\n'
assert s.count(old) == 1, 'm585'
p.write_text(s.replace(old, '\t\treturn func() { <-slot }, nil\n', 1))
