import pathlib
# 0.19.0 W10.3 (D-208, D-219): a pair nearer than the ground wave's reach is not drawn.
# Killed by TestEachPairIsGivenItsOutcome.
p = pathlib.Path('tools/reachscore/scenarios.go'); s = p.read_text()
old = 'if km < r.SkyKm {'
assert old in s, 'm467'
p.write_text(s.replace(old, 'if false {', 1))
