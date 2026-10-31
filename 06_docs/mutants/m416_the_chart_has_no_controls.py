import pathlib
# 0.19.0 AX-6: the chart window has a controls line.
# Killed by TestTheChartIsReadableInASCII.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\treturn append(out, "", d.propChartControls(o))\n'
assert old in s, 'm416'
p.write_text(s.replace(old, '\treturn out\n'))
