import pathlib
# 0.19.5 Q2.5 (RN-6): the operator's card carries the reports they ticked.
# Killed by TestTheOperatorsCardIsTheOperators.
p = pathlib.Path('app/mastercontrol.go'); s = p.read_text()
old = '\t\tSubject: key, Headline: ref.Label, Reports: kinds,\n'
assert s.count(old) == 1, 'm693'
p.write_text(s.replace(old, '\t\tSubject: key, Headline: ref.Label,\n', 1))
