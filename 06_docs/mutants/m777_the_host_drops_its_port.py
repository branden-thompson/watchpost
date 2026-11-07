import pathlib
# 0.19.5 Q5.6 (DR-16): a host keeps the port its URL names, as every copy did.
# Killed by TestHostOfNamesTheHostEveryCopyNamed.
p = pathlib.Path('platform/httpx/httpx.go'); s = p.read_text()
old = '\t\treturn rawURL\n\t}\n\treturn u.Host\n'
assert s.count(old) == 1, 'm777'
p.write_text(s.replace(old, '\t\treturn rawURL\n\t}\n\treturn u.Hostname()\n', 1))
