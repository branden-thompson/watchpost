import pathlib
# 0.19.3 AX-6, D-365: the Description chip says whether its box is open.
# Killed by TestEveryMapControlSaysItsState.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\tif d.mapPane.alertsOn {\n\t\t\tdescription = "Description On"\n'
assert s.count(old) == 1, 'm610'
p.write_text(s.replace(old, '\t\tif d.mapPane.alertsOn && false {\n\t\t\tdescription = "Description On"\n', 1))
