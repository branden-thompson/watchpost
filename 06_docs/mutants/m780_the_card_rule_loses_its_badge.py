import pathlib
# 0.19.5 Q6.2 (D-18): the card's rule, the masthead's rule, carries the grade
# badge at its right.
# Killed by TestTheBoxsRuleCarriesTheGrade.
p = pathlib.Path('modes/tty/broadcaster.go'); s = p.read_text()
old = '\treturn render.TitledRule(mark, title, badge, inner)\n'
assert s.count(old) == 1, 'm780'
p.write_text(s.replace(old, '\treturn render.TitledRule(mark, title, "", inner)\n', 1))
