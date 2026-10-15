import pathlib
# 0.19.0 D-100: the chart answers for the Setting's near-vertical radius.
# Killed by TestTheNearVerticalRadiusIsASetting.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = 'radiusKm: float64(d.propRadiusKm()), noise: shown.Noise}, true'
assert old in s, 'm195'
p.write_text(s.replace(old, 'radiusKm: 400, noise: shown.Noise}, true'))
