import pathlib
# 0.19.0 W10.3 (D-208, D-219): a stratum keeps at most k of each outcome.
# Killed by TestTheDrawIsBalancedSoAConstantScoresHalf.
p = pathlib.Path('tools/reachscore/scenarios.go'); s = p.read_text()
old = 'if len(r.kept) < k {'
assert old in s, 'm460'
p.write_text(s.replace(old, 'if len(r.kept) < k+1000 {', 1))
