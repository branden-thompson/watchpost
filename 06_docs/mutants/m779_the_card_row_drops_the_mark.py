import pathlib
# 0.19.5 Q6.1 (D-10): the card row, behind render, draws a fabricated event's
# mark in place of the inset.
# Killed by TestAFabricatedCardIsMarkedInTheLineUp.
p = pathlib.Path('platform/render/card_row.go'); s = p.read_text()
old = '\tif mark != "" {\n\t\trow, data["mark"] = r.marked, mark\n\t}\n'
assert s.count(old) == 1, 'm779'
p.write_text(s.replace(old, '\tif mark != "" && false {\n\t\trow, data["mark"] = r.marked, mark\n\t}\n', 1))
