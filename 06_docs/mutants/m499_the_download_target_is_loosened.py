import pathlib
# 0.19.0 W10.2: the downloads are held to ten times G1's target.
# Killed by TestAG1RunIsTheDifferenceOfItsArms.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = 'use.wireMB/use.wireHours <= wireTarget'
assert s.count(old) == 1, 'm499'
p.write_text(s.replace(old, 'use.wireMB/use.wireHours <= 10*wireTarget', 1))
