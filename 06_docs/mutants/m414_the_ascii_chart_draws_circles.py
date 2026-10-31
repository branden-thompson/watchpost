import pathlib
# 0.19.0 AX-4: under --ascii the chart draws letters.
# Killed by TestTheChartIsReadableInASCII.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tif d.cfg.ASCII {\n\t\treturn propASCIIMarks[g]\n'
assert old in s, 'm414'
p.write_text(s.replace(old, '\tif d.cfg.ASCII && false {\n\t\treturn propASCIIMarks[g]\n'))
