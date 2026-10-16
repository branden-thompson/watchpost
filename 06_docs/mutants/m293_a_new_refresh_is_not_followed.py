import pathlib
# 0.19.0 D-120: the open mode's tick follows its Setting.
# Killed by TestTheRefreshSettingStatesItsCost.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'd.mapPane.propTickEvery == d.mapPropRefresh.every() {'
assert old in s, 'm293'
p.write_text(s.replace(old, 'true {'))
