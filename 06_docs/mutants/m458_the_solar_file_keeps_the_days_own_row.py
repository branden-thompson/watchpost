import pathlib
# 0.19.0 W10.3 (D-208, D-221): the replay's solar file holds only the days before the clock's.
# Killed by TestTheReplayServesNothingAfterItsClock.
p = pathlib.Path('tools/reachscore/replay.go'); s = p.read_text()
old = 'if err != nil || !d.Before(today) {'
assert old in s, 'm458'
p.write_text(s.replace(old, 'if err != nil || d.After(today) {', 1))
