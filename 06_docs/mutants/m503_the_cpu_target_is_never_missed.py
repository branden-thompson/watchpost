import pathlib
# 0.19.0 W10.2: the added CPU always meets its target.
# Killed by TestAG1RunIsTheDifferenceOfItsArms.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = 'verdict(cpu <= g1AddedCPUPct)'
assert s.count(old) == 1, 'm503'
p.write_text(s.replace(old, 'verdict(true)', 1))
