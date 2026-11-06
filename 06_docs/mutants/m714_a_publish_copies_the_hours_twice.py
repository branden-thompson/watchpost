import pathlib
# 0.19.5 Q2.12 (A-15): a publish's bytes are pinned, so an extra copy of every
# location's hourly forecast cannot come back unseen.
# Killed by TestAssemblerPublishAllocBudget.
p = pathlib.Path('platform/snapshot/assembler.go'); s = p.read_text()
old = '\tcp.Hourly = append([]Hourly(nil), sec.Hourly...)\n'
assert s.count(old) == 1, 'm714'
p.write_text(s.replace(old, old + '\tcp.Hourly = append([]Hourly(nil), cp.Hourly...)\n', 1))
