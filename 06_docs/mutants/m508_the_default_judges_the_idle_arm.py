import pathlib
# 0.19.0 D-236: G1 is judged by default against the map never opened.
# Killed by TestAG1RunIsJudgedAgainstTheArmNamed.
p = pathlib.Path('tools/perfsum/main.go'); s = p.read_text()
old = 'fs.String("against", "map",'
assert s.count(old) == 1, 'm508'
p.write_text(s.replace(old, 'fs.String("against", "idle",', 1))
