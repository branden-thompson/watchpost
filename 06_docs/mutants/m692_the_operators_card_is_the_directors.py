import pathlib
# 0.19.5 Q2.5 (RN-6): the operator's card is minted as the operator's.
# Killed by TestTheOperatorsCardIsTheOperators.
p = pathlib.Path('app/mastercontrol.go'); s = p.read_text()
old = 'Slot: lineup.LocationReport, Origin: lineup.FromOperator,'
assert s.count(old) == 1, 'm692'
p.write_text(s.replace(old, 'Slot: lineup.LocationReport, Origin: lineup.FromDirector,', 1))
