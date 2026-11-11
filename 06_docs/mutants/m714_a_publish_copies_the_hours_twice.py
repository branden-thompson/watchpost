import pathlib
# 0.19.5 Q2.12 (A-15), Q4.1: a publish's bytes are pinned, so a copy of every
# location's hourly forecast cannot come back unseen once the hours are shared.
# Killed by TestAssemblerPublishAllocBudget.
p = pathlib.Path('platform/snapshot/assembler.go'); s = p.read_text()
old = '\t\t\tcp.Hourly = sec.Hourly\n'
assert s.count(old) == 1, 'm714'
p.write_text(s.replace(old, '\t\t\tcp.Hourly = append([]Hourly(nil), sec.Hourly...)\n', 1))
