import pathlib
# 0.19.0 W10.2: the driver's end-of-run counters record is left out.
# Killed by TestAG1RunIsJudgedAgainstTheArmNamed.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = "\t\traw = append(append(raw, '\\n'), end...)\n"
assert s.count(old) == 1, 'm506'
p.write_text(s.replace(old, '\t\t_ = end\n', 1))
