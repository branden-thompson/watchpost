import pathlib
# 0.19.0 FR-1.17: an answer to an older ask is dropped.
# Killed by TestAnswersNeverBlockTheUI.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tif v.chart.key != d.mapPane.propChartAsked {\n'
assert old in s, 'm192'
p.write_text(s.replace(old, '\tif false {\n'))
