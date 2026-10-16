import pathlib
# 0.19.0 D-181, D-198: a row whose nearest station is unscored is its ? shade.
# Killed by TestAnUnscoredNearestStationIsSaid.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tcase n.Unscored:\n\t\treturn "?"'
assert old in s, 'm223'
p.write_text(s.replace(old, '\tcase false:\n\t\treturn "?"'))
