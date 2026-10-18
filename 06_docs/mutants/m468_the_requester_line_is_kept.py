import pathlib
# 0.19.0 W10.3 (D-208, D-221): the replay leaves out the reply's line naming the requester.
# Killed by TestTheReplayServesNothingAfterItsClock.
p = pathlib.Path('tools/reachscore/replay.go'); s = p.read_text()
old = 'if strings.HasPrefix(text, "# Requested by") {'
assert old in s, 'm468'
p.write_text(s.replace(old, 'if false {', 1))
