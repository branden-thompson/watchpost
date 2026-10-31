import pathlib
# 0.19.0 W10.3 (D-208): the shipped build's column makes the stations' burst after NOAA's.
# Killed by TestEachColumnIsFedAsItSays.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = 'if err == nil && s.Reason == ionomaps.StationsNext {'
assert old in s, 'm466'
p.write_text(s.replace(old, 'if false {', 1))
