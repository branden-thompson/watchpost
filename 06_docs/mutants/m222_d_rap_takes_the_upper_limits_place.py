import pathlib
# 0.19.0 R-2.6: a row's words keep its upper limit and add D-RAP's.
# Killed by TestADisturbedBandIsDrawnAndNamed.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\tcase ionomaps.LimitDRAP:\n\t\t\tdrap = propLimitWords(b)\n'
assert old in s, 'm222'
p.write_text(s.replace(old, '\t\tcase ionomaps.LimitDRAP:\n\t\t\tdrap = \"\"\n'))
