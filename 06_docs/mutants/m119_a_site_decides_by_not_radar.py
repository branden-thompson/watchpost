import pathlib
# W2.0 (C-M2): a site that says "not Forecast" also means every mode added later.
# Killed by TestEveryMapModeIsHandledAtEverySite.
p = pathlib.Path("modes/tty/map_temp.go"); s = p.read_text()
old = "\tif d.mapMode() == modeRadar || d.mapMode() == modePropagation {\n\t\treturn d\n\t}\n\td.mapPane.fcLow = !d.mapPane.fcLow"
assert old in s, "m119"
p.write_text(s.replace(old, "\tif d.mapMode() != modeForecast {\n\t\treturn d\n\t}\n\td.mapPane.fcLow = !d.mapPane.fcLow"))
