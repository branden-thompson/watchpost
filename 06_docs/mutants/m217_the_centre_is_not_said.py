import pathlib
# 0.19.0 FR-1.11, FR-3.1: the centre is said in the words, last.
# Killed by TestTheCentreAnswerIsSaid.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = '\t\t\tout = append(out, c) // and the centre, last (FR-1.11, FR-3.1)\n'
assert old in s, 'm217'
p.write_text(s.replace(old, '\t\t\t_ = c\n'))
