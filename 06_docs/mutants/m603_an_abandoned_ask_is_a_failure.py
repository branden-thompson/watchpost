import pathlib
# 0.19.1 D-354: an ask its caller cancelled is a try and not the host failing.
# Killed by TestAnAbandonedAskIsNotAFailure.
p = pathlib.Path('platform/httpx/httpx.go'); s = p.read_text()
old = '\tdefault:\n\t\tif !Abandoned(ctx) {\n'
assert s.count(old) == 1, 'm603'
p.write_text(s.replace(old, '\tdefault:\n\t\tif !Abandoned(ctx) || true {\n', 1))
