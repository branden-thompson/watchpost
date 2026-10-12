import pathlib
# 0.19.0 FR-1.17 (A-27): the chart is asked once for what it is worked out from.
# Killed by TestAnswersNeverBlockTheUI.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tif !ok || key == d.mapPane.propChart.key || key == d.mapPane.propChartAsked {\n'
assert old in s, 'm191'
p.write_text(s.replace(old, '\tif !ok {\n'))
