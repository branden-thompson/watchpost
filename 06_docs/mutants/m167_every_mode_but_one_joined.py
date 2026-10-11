import pathlib
# C-M2: every mode but one, joined with ||, also means every mode added later.
# Killed by TestEveryMapModeIsHandledAtEverySite.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\tm := d.mapPane.m\n\tif m == nil {\n\t\treturn d\n\t}\n\tif d.mapMode() == modePropagation {\n\t\tleast'
assert old in s, 'm167'
p.write_text(s.replace(old, '\tm := d.mapPane.m\n\tif m == nil || d.mapMode() == modeRadar || d.mapMode() == modeForecast {\n\t\treturn d\n\t}\n\tif d.mapMode() == modePropagation {\n\t\tleast'))
