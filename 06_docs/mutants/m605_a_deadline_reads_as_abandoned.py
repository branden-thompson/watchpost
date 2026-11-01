import pathlib
# 0.19.1 D-354: an ask past its deadline is the host not answering, a failure.
# Killed by TestAnAbandonedAskIsNotAFailure, TestAnAbandonedPlainAskIsNotAFailure and TestAnAbandonedTileIsNotAFailure.
p = pathlib.Path('platform/httpx/stats.go'); s = p.read_text()
old = '\treturn errors.Is(ctx.Err(), context.Canceled)\n'
assert s.count(old) == 1, 'm605'
p.write_text(s.replace(old, '\treturn ctx.Err() != nil || errors.Is(ctx.Err(), context.Canceled)\n', 1))
