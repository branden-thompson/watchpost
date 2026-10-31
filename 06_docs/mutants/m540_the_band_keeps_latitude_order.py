import pathlib
# 0.19.0 RP-6: the band is read in latitude order, so places at one distance rank otherwise than the whole scan ranks them.
# Killed by TestTheBandKeepsTheTablesOrder.
p = pathlib.Path('domains/locations/geodata/near.go'); s = p.read_text()
old = '\tslices.Sort(band) //'
assert s.count(old) == 1, 'm540'
p.write_text(s.replace(old, '\t_ = band //', 1))
