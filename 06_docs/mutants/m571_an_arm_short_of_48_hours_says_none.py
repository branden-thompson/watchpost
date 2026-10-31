import pathlib
# 0.19.0 D-273 (R4D-4): an arm short of G1's 48 hours prints no shortfall, so the summary cannot show its margin.
# Killed by TestAG1RunThatCannotBeJudgedFails.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = '\t\tif a.hours < 48 {\n'
assert s.count(old) == 1, 'm571'
p.write_text(s.replace(old, '\t\tif a.hours < 0 {\n', 1))
