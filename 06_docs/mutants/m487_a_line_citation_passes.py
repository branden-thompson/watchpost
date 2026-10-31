import pathlib
# 0.19.0 BUILD red team round 2 (CN2-7, DN2-8): a living document's line citation is refused; it cites the symbol.
# Killed by TestEveryCodeCitationResolves.
p = pathlib.Path('cmd/watchpost/record_test.go'); s = p.read_text()
old = '\t\tif m[1] == "" {\n\t\t\tout = append(out, m[2]+":"+m[3]+" (cite its symbol)")'
assert old in s, 'm487'
p.write_text(s.replace(old, '\t\tif false {\n\t\t\tout = append(out, m[2]+":"+m[3]+" (cite its symbol)")', 1))
