import pathlib
# 0.19.5 Q7.4 (JM-3): extending.md's `pkg.Symbol` resolves only to what that package declares.
# Killed by TestTheSymbolCheckRefusesWhatIsNotDeclared.
p = pathlib.Path('cmd/watchpost/extending_symbols_test.go'); s = p.read_text()
old = '\t\tif ix.top[dir][parts[1]] && ix.membersOf(parts[1], parts[2:]) {\n'
assert s.count(old) == 1, 'm788'
p.write_text(s.replace(old, '\t\tif _ = dir; ix.membersOf(parts[1], parts[2:]) {\n', 1))
