import pathlib
# 0.19.0 FR-5.4: the chart's status label says the offset's state.
# Killed by TestAnUnusualOffsetIsSaid.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\tout[1] = propOffsetNote(s.Offset)'
assert old in s, 'm285'
p.write_text(s.replace(old, '\t\tout[1] = ""'))
