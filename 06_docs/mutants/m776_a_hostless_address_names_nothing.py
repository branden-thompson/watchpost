import pathlib
# 0.19.5 Q5.6 (DR-16): an address with no host names itself, never "".
# Killed by TestHostOfNamesTheHostEveryCopyNamed.
p = pathlib.Path('platform/httpx/httpx.go'); s = p.read_text()
old = '\tif err != nil || u.Host == "" {\n\t\treturn rawURL\n\t}\n\treturn u.Host\n'
assert s.count(old) == 1, 'm776'
p.write_text(s.replace(old, '\tif err != nil {\n\t\treturn rawURL\n\t}\n\treturn u.Host\n', 1))
