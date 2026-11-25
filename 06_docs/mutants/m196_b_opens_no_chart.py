import pathlib
# 0.19.0 D-176: B opens the chart over the map.
# Killed by TestBestBandsNameTheirLimit.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\tmm = mm.flashMapKey(act, e)\n\t\treturn mm, nil, mm.openPropChart(e), true\n'
assert old in s, 'm196'
p.write_text(s.replace(old, '\t\tmm = mm.flashMapKey(act, e)\n\t\treturn mm, nil, mapWant{}, true\n'))
