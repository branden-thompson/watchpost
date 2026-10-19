import pathlib
# 0.19.0 D-239: the D-RAP and scales the days never hold count against the run.
# Killed by TestOnlyAFileTheReplayServesFailsTheRun.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = '\t\tif !slices.Contains(neverRecorded, f) && !withheld {\n'
assert s.count(old) == 1, 'm509'
p.write_text(s.replace(old, '\t\tif !slices.Contains(neverRecorded[:0], f) && !withheld {\n', 1))
