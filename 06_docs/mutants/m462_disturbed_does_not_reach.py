import pathlib
# 0.19.0 W10.3 (D-208, D-220): a disturbed band reaches (C1), as the product reads it.
# Killed by TestTheAnswerIsReadAsTheProductReadsIt.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = 'return len(propanswers.Usable([]ionomaps.BandStatus{b})) == 1'
assert old in s, 'm462'
p.write_text(s.replace(old, 'return b.Status == ionomaps.Open', 1))
