import pathlib
# 0.19.0 D-324: a host's share is the whole lane, so one that hangs holds every slot.
# Killed by TestAHungHostHoldsAtMostHalfALane.
p = pathlib.Path('platform/httpx/httpx.go'); s = p.read_text()
old = '\t\ts[l] = make(chan struct{}, cap(c.inflight[l])/2)\n'
assert s.count(old) == 1, 'm584'
p.write_text(s.replace(old, '\t\ts[l] = make(chan struct{}, cap(c.inflight[l]))\n', 1))
