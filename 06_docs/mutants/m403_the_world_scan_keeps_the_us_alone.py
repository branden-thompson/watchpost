import pathlib
# 0.19.0 AX-1: the world scan reads every country, not the US alone.
# Killed by TestAWorldScanAnswersAsTheTableDoes.
p = pathlib.Path('domains/locations/geodata/near.go'); s = p.read_text()
old = 'func (i *Index) worldCities() []placed { return i.placedOnce(&i.worldOnce, &i.world, false) }'
assert old in s, 'm403'
p.write_text(s.replace(old, 'func (i *Index) worldCities() []placed { return i.placedOnce(&i.worldOnce, &i.world, true) }'))
