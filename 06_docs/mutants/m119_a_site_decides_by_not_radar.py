import pathlib
# W2.0 (C-M2): a site that says "not Forecast" also means every mode added later.
# Killed by TestEveryMapModeIsHandledAtEverySite.
p = pathlib.Path("modes/tty/map_temp.go"); s = p.read_text()
old = '\tif mm.mapMode(e) == modeForecast {\n\t\tmm.mapPane.fcLow = !mm.mapPane.fcLow'
assert old in s, "m119"
p.write_text(s.replace(old, '\tif mm.mapMode(e) != modeRadar && mm.mapMode(e) != modePropagation {\n\t\tmm.mapPane.fcLow = !mm.mapPane.fcLow'))
