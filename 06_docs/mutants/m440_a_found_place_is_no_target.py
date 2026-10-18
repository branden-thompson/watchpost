import pathlib
# 0.19.0 D-204: on the Observer a place found joins the chart as a target row.
# Killed by TestAFoundPlaceIsAnsweredAndNotKept.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tif f := d.mapPane.propFound; f != nil && !d.propRole().target {\n'
assert old in s, 'm440'
p.write_text(s.replace(old, '\tif f := d.mapPane.propFound; f != nil && false {\n'))
