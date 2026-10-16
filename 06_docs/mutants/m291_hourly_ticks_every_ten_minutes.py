import pathlib
# 0.19.0 D-120: hourly is hourly.
# Killed by TestTheRefreshSettingStatesItsCost.
p = pathlib.Path('modes/tty/map_prop_settings.go'); s = p.read_text()
old = '\tcase refreshHourly:\n\t\treturn time.Hour\n'
assert old in s, 'm291'
p.write_text(s.replace(old, '\tcase refreshHourly:\n\t\treturn 10 * time.Minute\n'))
