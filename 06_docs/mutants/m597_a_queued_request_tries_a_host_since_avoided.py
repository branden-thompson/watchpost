import pathlib
# 0.19.0 VALIDATE VC-2: a request queued on a hung host's share still tries it once the memo has armed.
# Killed by TestRequestsQueuedOnAHungHostFailFast.
p = pathlib.Path('platform/httpx/httpx.go'); s = p.read_text()
old = '\tif err := c.memoRefusal(req); err != nil {\n\t\tc.stats.add(req.host, func(h *HostStats) { h.FastFail++; h.LastFail = time.Now() })\n\t\treturn nil, nil, err\n\t}\n\tvar last outcome\n'
assert s.count(old) == 1, 'm597'
p.write_text(s.replace(old, '\tvar last outcome\n', 1))
