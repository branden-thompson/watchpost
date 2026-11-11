import pathlib
# 0.19.5 Q4.5 (D-22): the line-up generation is process-wide, so two copies sharing a memo never share it.
# Killed by TestTheConsolesProjectionFollowsItsLineup.
p = pathlib.Path('modes/tty/broadcaster.go'); s = p.read_text()
old = 'b.lineup, b.lineupGen = v.Lineup, lineupGens.Add(1)'
assert s.count(old) == 1, 'm754'
p.write_text(s.replace(old, 'b.lineup, b.lineupGen = v.Lineup, b.lineupGen+1', 1))
