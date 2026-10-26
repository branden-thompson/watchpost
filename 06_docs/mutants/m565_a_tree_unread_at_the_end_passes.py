import pathlib
# 0.19.0 RC-3 (R3C-8): a green run whose tree cannot be read again at its end exits 0.
# Killed by TestARunWithNoRecordIsNoPass.
p = pathlib.Path('tools/runlog/main.go'); s = p.read_text()
old = '\ttree, err := testedTree(dir)\n\tif err != nil {\n\t\treturn max(rc, 1), err\n'
assert s.count(old) == 1, 'm565'
p.write_text(s.replace(old, '\ttree, err := testedTree(dir)\n\tif err != nil {\n\t\treturn rc, err\n', 1))
