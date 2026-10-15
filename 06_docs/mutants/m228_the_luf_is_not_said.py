import pathlib
# 0.19.0 D-192: a row's words add the LUF.
# Killed by TestAClosedBandIsDrawnByItsKind.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\t\tlower = propLimitWords(b)\n'
assert old in s, 'm228'
p.write_text(s.replace(old, '\t\t\tlower = ""\n'))
