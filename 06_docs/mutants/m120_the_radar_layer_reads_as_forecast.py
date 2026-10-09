import pathlib
# W2.0 (D-94): the radar layer on is Radar mode; read the other way, R switches nothing it says.
# Killed by TestForecastModeDrawsTheAirsDays, TestTheBadgeRowCreditsEachLayerDrawn, TestWavesAreTheirOwnRow and more.
p = pathlib.Path("modes/tty/map_temp.go"); s = p.read_text()
old = "\tif d.chosen(RadarLayer) {\n\t\treturn modeRadar\n\t}\n\treturn modeForecast"
assert old in s, "m120"
p.write_text(s.replace(old, "\tif d.chosen(RadarLayer) {\n\t\treturn modeForecast\n\t}\n\treturn modeRadar"))
