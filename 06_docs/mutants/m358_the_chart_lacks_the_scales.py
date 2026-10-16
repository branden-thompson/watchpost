import pathlib
# 0.19.0 FR-1.18: the scales are named in the chart.
# Killed by TestARaisedScaleIsNamed.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\tif l := propScalesLine(s.Scales); l != "" {\n'
assert old in s, 'm358'
p.write_text(s.replace(old, '\t\tif l := propScalesLine(s.Scales); false && l != "" {\n'))
