import pathlib
# 0.19.0 W10.3 (D-208): the climatology's column is never given a GloTEC grid.
# Killed by TestAColumnFedOtherwiseFailsTheRun.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = 'return ionomapstest.Options{WithoutGloTEC: col == climatology}'
assert old in s, 'm464'
p.write_text(s.replace(old, 'return ionomapstest.Options{}', 1))
