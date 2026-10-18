import pathlib
# 0.19.0 W10.3 (D-208, D-221): the replay serves no GIRO reading taken after its clock.
# Killed by TestTheReplayServesNothingAfterItsClock.
p = pathlib.Path('tools/reachscore/replay.go'); s = p.read_text()
old = 'if at.Before(from) || at.After(to) || at.After(now) {'
assert old in s, 'm459'
p.write_text(s.replace(old, 'if at.Before(from) || at.After(to) {', 1))
