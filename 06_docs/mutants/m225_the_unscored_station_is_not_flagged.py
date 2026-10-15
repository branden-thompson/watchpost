import pathlib
# 0.19.0 D-181: the centre says to verify on air where its station is unscored.
# Killed by TestAnUnscoredNearestStationIsSaid.
p = pathlib.Path('modes/tty/map_prop_centre.go'); s = p.read_text()
old = '\tif n.Unscored {\n'
assert old in s, 'm225'
p.write_text(s.replace(old, '\tif n.Unscored && false {\n'))
