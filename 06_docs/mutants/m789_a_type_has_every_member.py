import pathlib
# 0.19.5 Q7.4 (JM-3): extending.md's `Type.Member` resolves only to that type's own fields and methods.
# Killed by TestTheSymbolCheckRefusesWhatIsNotDeclared.
p = pathlib.Path('cmd/watchpost/extending_symbols_test.go'); s = p.read_text()
old = '\tif !ix.members[typ][chain[0]] {\n'
assert s.count(old) == 1, 'm789'
p.write_text(s.replace(old, '\tif !ix.any[chain[0]] {\n', 1))
