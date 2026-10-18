import pathlib
# 0.19.0 W10.3 (D-208, D-218): an answer agrees when it reaches on an open pair or does not on a closed one.
# Killed by TestTheAnswerIsReadAsTheProductReadsIt.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = 'if reached == (o == open) {'
assert old in s, 'm463'
p.write_text(s.replace(old, 'if reached != (o == open) {', 1))
