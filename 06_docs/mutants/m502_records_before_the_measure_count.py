import pathlib
# 0.19.0 W10.2: counters records from before the measure are counted.
# Killed by TestAG1RunIsTheDifferenceOfItsArms.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = '\t\tif r.At.Before(from) {\n\t\t\tcontinue\n\t\t}\n'
assert s.count(old) == 1, 'm502'
p.write_text(s.replace(old, '', 1))
