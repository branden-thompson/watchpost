import pathlib
# 0.19.0 D-121: the refresh is one Setting for both UI modes.
# Killed by TestTheConsolePropagationModeHonoursTheSharedSettings.
p = pathlib.Path('modes/tty/setup_rows.go'); s = p.read_text()
old = 'rowMapPropRefresh:  {rowMapPropRefresh, groupMap, scopeShared,'
assert old in s, 'm297'
p.write_text(s.replace(old, 'rowMapPropRefresh:  {rowMapPropRefresh, groupMap, scopeObserver,'))
