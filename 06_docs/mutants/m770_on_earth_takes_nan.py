import pathlib
# 0.19.5 Q5.6 (DR-9): OnEarth is written so NaN fails it; the negative form
# two of its copies used takes NaN.
# Killed by TestOnEarthRefusesWhatIsNoPlace.
p = pathlib.Path('platform/geo/coords.go'); s = p.read_text()
old = '\treturn lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180\n'
assert s.count(old) == 1, 'm770'
p.write_text(s.replace(old, '\treturn !(lat < -90 || lat > 90 || lon < -180 || lon > 180)\n', 1))
