import pathlib
# 0.19.0 RC-3: a green run whose line cannot be written exits 0, a pass with no record.
# Killed by TestARunWithNoRecordIsNoPass.
p = pathlib.Path('tools/runlog/main.go'); s = p.read_text()
old = '\tf, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)\n\tif err != nil {\n\t\treturn max(rc, 1), err\n'
assert s.count(old) == 1, 'm556'
p.write_text(s.replace(old, '\tf, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)\n\tif err != nil {\n\t\treturn rc, err\n', 1))
