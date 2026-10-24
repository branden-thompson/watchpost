import pathlib
# 0.19.0 RC-1: a default run under 48 hours exits as met.
# Killed by TestAG1RunThatCannotBeJudgedFails.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = 'exits as one missed (RC-1).\\n")\n\t\tmissed++\n'
assert s.count(old) == 1, 'm528'
p.write_text(s.replace(old, 'exits as one missed (RC-1).\\n")\n', 1))
