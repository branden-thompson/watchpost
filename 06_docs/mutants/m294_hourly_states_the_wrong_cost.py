import pathlib
# 0.19.0 D-141: hourly's measured cost.
# Killed by TestTheRefreshSettingStatesItsCost.
p = pathlib.Path('modes/tty/map_prop_settings.go'); s = p.read_text()
old = '"up to 3.3 MB an hour while the Propagation mode is open"'
assert old in s, 'm294'
p.write_text(s.replace(old, '"up to 19 MB an hour while the Propagation mode is open"'))
