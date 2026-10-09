import pathlib
# W2.0 (C-M2): a site that says "not Radar" also means every mode added later.
# Killed by TestEveryMapModeIsHandledAtEverySite.
p = pathlib.Path("modes/tty/map_pane.go"); s = p.read_text()
old = "\tif d.mapMode() == modeForecast { // Forecast mode: the host steps (D-94)"
assert old in s, "m119"
p.write_text(s.replace(old, "\tif d.mapMode() != modeRadar { // Forecast mode: the host steps (D-94)"))
