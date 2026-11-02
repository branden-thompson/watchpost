import pathlib
# 0.19.3 D-361: Forecast mode holds its last day 2 s whatever the speed.
# Killed by TestForecastStepsAtThePlaybackSpeed.
p = pathlib.Path('modes/tty/map_temp.go'); s = p.read_text()
old = "\tcase d.mapPane.fcHeld < forecastHoldSteps(d.mapPlayback)-1:"
assert s.count(old) == 1, 'm624'
p.write_text(s.replace(old, "\tcase d.mapPane.fcHeld < 1:", 1))
