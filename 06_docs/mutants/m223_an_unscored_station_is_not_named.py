import pathlib
# 0.19.0 D-181: the chart names the rows whose nearest station is unscored.
# Killed by TestAnUnscoredNearestStationIsSaid.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\tif a.Nearest.Known && a.Nearest.Unscored {\n'
assert old in s, 'm223'
p.write_text(s.replace(old, '\t\tif a.Nearest.Known && a.Nearest.Unscored && false {\n'))
