import pathlib
# 0.19.0 D-332: a file naming no refresh reads as every 10 minutes, not hourly.
# Killed by TestTheRefreshSettingStatesItsCost.
p = pathlib.Path('modes/tty/map_prop_settings.go'); s = p.read_text()
old = '\t\t}\n\t}\n\treturn refreshHourly\n}\n'
assert s.count(old) == 1, 'm590'
p.write_text(s.replace(old, '\t\t}\n\t}\n\treturn refreshEvery10\n}\n', 1))
