import pathlib
# 0.19.0 D-120: on demand, no tick is set.
# Killed by TestTheRefreshSettingStatesItsCost.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif every == 0 {\n\t\treturn d, nil\n'
assert old in s, 'm292'
p.write_text(s.replace(old, '\tif false {\n\t\treturn d, nil\n'))
