import pathlib
# 0.19.0 D-120: hourly is hourly.
# Killed by TestTheRefreshSettingStatesItsCost.
p = pathlib.Path('modes/tty/map_prop_settings.go'); s = p.read_text()
old = '\t\treturn 0\n\t}\n\treturn time.Hour\n}\n'
assert old in s, 'm291'
p.write_text(s.replace(old, '\t\treturn 0\n\t}\n\treturn 10 * time.Minute\n}\n'))
