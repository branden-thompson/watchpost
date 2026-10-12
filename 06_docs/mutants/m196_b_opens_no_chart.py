import pathlib
# 0.19.0 D-176: B opens the chart over the map.
# Killed by TestBestBandsNameTheirLimit.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\treturn d.flashMapKey(act).openPropChart(), nil, true\n'
assert old in s, 'm196'
p.write_text(s.replace(old, '\t\treturn d.flashMapKey(act), nil, true\n'))
