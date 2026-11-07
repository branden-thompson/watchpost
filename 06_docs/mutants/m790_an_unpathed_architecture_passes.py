import pathlib
# 0.19.5 Q7.4 (JM-13): an "architecture §n" citation names its architecture.md.
# Killed by TestTheSymbolCheckRefusesWhatIsNotDeclared.
p = pathlib.Path('cmd/watchpost/extending_symbols_test.go'); s = p.read_text()
old = 'architectureCite = regexp.MustCompile("(?i)architecture(?:\\\\.md)?`?\\\\s*§")'
assert s.count(old) == 1, 'm790'
p.write_text(s.replace(old, 'architectureCite = regexp.MustCompile("(?i)architecture\\\\.md`?\\\\s*§")', 1))
