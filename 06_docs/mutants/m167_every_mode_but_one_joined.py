import pathlib
# C-M2: every mode but one, joined with ||, also means every mode added later.
# Killed by TestEveryMapModeIsHandledAtEverySite.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\tm := mm.mapPane.m\n\tif m == nil {\n\t\treturn mm\n\t}\n\tif mm.mapMode(e) == modePropagation {\n\t\tleast'
assert old in s, 'm167'
p.write_text(s.replace(old, '\tm := mm.mapPane.m\n\tif m == nil || mm.mapMode(e) == modeRadar || mm.mapMode(e) == modeForecast {\n\t\treturn mm\n\t}\n\tif mm.mapMode(e) == modePropagation {\n\t\tleast'))
