import pathlib
# 0.19.0 AX-8: the centre's clauses begin in the sentence's lower case.
# Killed by TestTheCentreAnswerIsSaid.
p = pathlib.Path('modes/tty/map_prop_centre.go'); s = p.read_text()
old = '\t\t\t\tlines[i] = strings.ToLower(lead[:1]) + l[1:]\n'
assert old in s, 'm424'
p.write_text(s.replace(old, '\t\t\t\tlines[i] = l\n'))
