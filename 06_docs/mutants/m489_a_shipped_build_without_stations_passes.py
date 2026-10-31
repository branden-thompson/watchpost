import pathlib
# 0.19.0 BUILD red team round 2 (CN2-9): a shipped column no station reached fails the run.
# Killed by TestAColumnFedOtherwiseFailsTheRun.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = '\tif col == shipped && stations == 0 {'
assert old in s, 'm489'
p.write_text(s.replace(old, '\tif col == shipped && stations < 0 {', 1))
