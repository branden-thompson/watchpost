import pathlib
# 0.19.0 W10.2: counters records any time apart are read as a rate.
# Killed by TestAG1RunIsTheDifferenceOfItsArms.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = '\ta.wireRead = a.wireHours >= 1\n'
assert s.count(old) == 1, 'm500'
p.write_text(s.replace(old, '\ta.wireRead = a.wireHours > 0\n', 1))
