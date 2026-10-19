import pathlib
# 0.19.0 W10.2 (D-141): an arm that never opened the mode may read the wire.
# Killed by TestAG1RunIsTheDifferenceOfItsArms.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = 'verdict(base.arm.wireRead && base.arm.wireMB == 0, counts)'
assert s.count(old) == 1, 'm501'
p.write_text(s.replace(old, 'verdict(base.arm.wireRead, counts)', 1))
