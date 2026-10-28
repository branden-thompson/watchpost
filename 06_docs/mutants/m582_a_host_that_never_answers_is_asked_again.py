import pathlib
# 0.19.0 D-324: the client's own timeout counted as the caller's, so a host that hangs is never avoided.
# Killed by TestAHostThatNeverAnswersIsAvoided.
p = pathlib.Path('platform/httpx/memo.go'); s = p.read_text()
old = '\t\tif ctx.Err() != nil {\n'
assert s.count(old) == 1, 'm582'
p.write_text(s.replace(old, '\t\tif ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) {\n', 1))
