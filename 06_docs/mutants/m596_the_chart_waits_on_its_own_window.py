import pathlib
# 0.19.0 batch 80: a chart opened over a settling view waits for a settle that waits for it.
# Killed by TestAChartOpenedAsTheViewMovesIsWorkedOut.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tif d.viewSettling() && d.propRole().origin == propFromCentre && d.modal != modalPropChart {\n'
assert s.count(old) == 1, 'm596'
p.write_text(s.replace(old, '\tif d.viewSettling() && d.propRole().origin == propFromCentre {\n', 1))
