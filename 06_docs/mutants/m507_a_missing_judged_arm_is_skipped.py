import pathlib
# 0.19.0 W10.2 (A-80): a run lacking the judged arm is summarised without it.
# Killed by TestAG1RunIsJudgedAgainstTheArmNamed.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = 'errors.Is(err, os.ErrNotExist) && base.key != against'
assert s.count(old) == 1, 'm507'
p.write_text(s.replace(old, 'errors.Is(err, os.ErrNotExist)', 1))
