import pathlib
# 0.19.0 W10.3 (D-208): a snapshot fed other than its column says fails the run.
# Killed by TestAColumnFedOtherwiseFailsTheRun.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = 'if gridWrong || stationWrong || f.Unread > 0 {'
assert old in s, 'm465'
p.write_text(s.replace(old, 'if (gridWrong || stationWrong || f.Unread > 0) && t.IsZero() {', 1))
