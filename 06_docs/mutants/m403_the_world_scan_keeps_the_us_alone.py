import pathlib
# 0.19.0 AX-1: the world scan reads every country, not the US alone.
# Killed by TestAWorldScanAnswersAsTheTableDoes.
p = pathlib.Path('domains/locations/geodata/near.go'); s = p.read_text()
old = '\t\ti.world = i.placedRows(false)\n'
assert old in s, 'm403'
p.write_text(s.replace(old, '\t\ti.world = i.placedRows(true)\n'))
