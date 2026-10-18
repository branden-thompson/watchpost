import pathlib
# 0.19.0 BUILD red team round 2 (CN2-9): a background or shipped column fed without its grid fails the run.
# Killed by TestAColumnFedOtherwiseFailsTheRun.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = '\t\tgridWrong := (col == climatology) == f.Grid'
assert old in s, 'm488'
p.write_text(s.replace(old, '\t\tgridWrong := col == climatology && f.Grid', 1))
