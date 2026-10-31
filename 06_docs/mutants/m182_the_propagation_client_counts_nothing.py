import pathlib
# D-169: the propagation client counts each ask on its host.
# Killed by TestThePropagationClientCountsItsRequests.
p = pathlib.Path('platform/httpx/plain.go'); s = p.read_text()
old = '\tp.stats.add(host, func(h *HostStats) { h.Attempts++ })\n'
assert old in s, 'm182'
p.write_text(s.replace(old, ''))
