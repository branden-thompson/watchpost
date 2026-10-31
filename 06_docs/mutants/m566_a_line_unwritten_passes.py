import pathlib
# 0.19.0 RC-3 (R3C-8): a green run whose line could not be written or closed exits 0.
# Killed by TestARunWithNoRecordIsNoPass.
p = pathlib.Path('tools/runlog/main.go'); s = p.read_text()
old = '\tif err = errors.Join(err, f.Close()); err != nil {\n\t\treturn max(rc, 1), err\n'
assert s.count(old) == 1, 'm566'
p.write_text(s.replace(old, '\tif err = errors.Join(err, f.Close()); err != nil {\n\t\treturn rc, err\n', 1))
