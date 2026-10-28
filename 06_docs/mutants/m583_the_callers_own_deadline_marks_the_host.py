import pathlib
# 0.19.0 D-324: a request its caller ended puts the host on the memo.
# Killed by TestTheCallersOwnDeadlineNeverMarksTheHost.
p = pathlib.Path('platform/httpx/memo.go'); s = p.read_text()
old = '\t\tif ctx.Err() != nil {\n'
assert s.count(old) == 1, 'm583'
p.write_text(s.replace(old, '\t\tif ctx.Err() != nil && ctx == nil {\n', 1))
