import pathlib
# 0.19.5 Q2.12 (A-15): the frame after a publish is pinned, so a miss that
# renders the tables twice cannot pass.
# Killed by TestPublishAllocBudget.
p = pathlib.Path('modes/tty/memo.go'); s = p.read_text()
old = '\tpriority, recent = d.priorityTable(fl), d.recentSection(fl)\n'
assert s.count(old) == 1, 'm715'
p.write_text(s.replace(old, '\t_, _ = d.priorityTable(fl), d.recentSection(fl)\n' + old, 1))
