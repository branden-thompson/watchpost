import pathlib
# 0.19.0 W10.3 (D-208, D-221): the replay's index names no grid valid after its clock.
# Killed by TestTheReplayServesNothingAfterItsClock.
p = pathlib.Path('tools/reachscore/replay.go'); s = p.read_text()
old = 'if !valid.After(r.now) {'
assert old in s, 'm457'
p.write_text(s.replace(old, 'if !valid.After(r.now.Add(24 * time.Hour)) {', 1))
