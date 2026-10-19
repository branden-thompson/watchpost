import pathlib
# 0.19.0 D-239: the climatology column's withheld GloTEC counts against the run.
# Killed by TestOnlyAFileTheReplayServesFailsTheRun.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = '\t\twithheld := col == climatology && f == "GloTEC"\n'
assert s.count(old) == 1, 'm510'
p.write_text(s.replace(old, '\t\twithheld := col == climatology && f == "none"\n', 1))
