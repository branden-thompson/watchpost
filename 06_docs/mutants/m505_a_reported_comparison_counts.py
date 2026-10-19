import pathlib
# 0.19.0 W10.2 (A-80): the comparison not judged counts towards the exit.
# Killed by TestAG1RunIsJudgedAgainstTheArmNamed.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = '\t\tcase counts:\n'
assert s.count(old) == 1, 'm505'
p.write_text(s.replace(old, '\t\tdefault:\n', 1))
