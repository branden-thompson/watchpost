import pathlib
# 0.19.0 W10.2: an arm is measured from its first phase, the open's cost in it.
# Killed by TestAG1RunIsTheDifferenceOfItsArms.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = '\t\tif strings.HasPrefix(p.name, g1Measured) {\n'
assert s.count(old) == 1, 'm504'
p.write_text(s.replace(old, '\t\tif true {\n', 1))
