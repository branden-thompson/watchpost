import pathlib
# 0.19.0 W10.2 (D-141): the arm that never opened the mode may read the wire.
# Killed by TestAG1RunIsTheDifferenceOfItsArms.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = 'verdict(idle.wireRead && idle.wireMB == 0)'
assert s.count(old) == 1, 'm501'
p.write_text(s.replace(old, 'verdict(idle.wireRead)', 1))
