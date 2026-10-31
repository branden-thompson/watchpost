import pathlib
# 0.19.0 D-324: a wait for a lane slot that its caller ends keeps the host's share.
# Killed by TestAWaitThatEndsGivesTheShareBack.
p = pathlib.Path('platform/httpx/httpx.go'); s = p.read_text()
old = '\tcase <-ctx.Done():\n\t\t<-share\n\t\treturn nil, ctx.Err()\n'
assert s.count(old) == 1, 'm586'
p.write_text(s.replace(old, '\tcase <-ctx.Done():\n\t\treturn nil, ctx.Err()\n', 1))
