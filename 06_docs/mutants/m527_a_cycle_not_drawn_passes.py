import pathlib
# 0.19.0 RC-1: a cycle the driver could not confirm drawn is no miss.
# Killed by TestAG1RunThatCannotBeJudgedFails.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = '\t\t\ta.warned++ // a cycle'
assert s.count(old) == 1, 'm527'
p.write_text(s.replace(old, '\t\t\t_ = a.warned // a cycle', 1))
