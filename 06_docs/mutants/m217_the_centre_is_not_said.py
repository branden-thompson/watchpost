import pathlib
# 0.19.0 FR-1.11, FR-3.1: the centre is said in the words, last.
# Killed by TestTheCentreAnswerIsSaid.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\tout = append(out, said("centre", spoken(c)))\n'
assert old in s, 'm217'
p.write_text(s.replace(old, '\t\t_ = c\n'))
