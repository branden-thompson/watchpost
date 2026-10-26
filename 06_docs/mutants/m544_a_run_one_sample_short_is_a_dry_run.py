import pathlib
# 0.19.0 D-273: an arm short of 48 hours by one 2-minute sample is a dry run.
# Killed by TestAG1RunThatCannotBeJudgedFails.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = 'const g1FullHours = 48 - 2.0/60\n'
assert s.count(old) == 1, 'm544'
p.write_text(s.replace(old, 'const g1FullHours = 48\n', 1))
