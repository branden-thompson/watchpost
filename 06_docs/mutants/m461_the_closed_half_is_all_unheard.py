import pathlib
# 0.19.0 W10.3 (D-208, D-218): the closed half is half weak and half unheard as far as each goes.
# Killed by TestTheDrawIsBalancedSoAConstantScoresHalf.
p = pathlib.Path('tools/reachscore/scenarios.go'); s = p.read_text()
old = 'nw := min(len(w), (n+1)/2)'
assert old in s, 'm461'
p.write_text(s.replace(old, 'nw := 0', 1))
