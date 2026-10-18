import pathlib
# 0.19.0 L-F12: an origin whose latitude is not a number is refused, not only one past the poles.
# Killed by TestAnAskOffTheGlobeIsRefused.
p = pathlib.Path('platform/propanswers/answers.go'); s = p.read_text()
old = '\tif !(ask.Origin.Lat >= -90 && ask.Origin.Lat <= 90) {\n\t\treturn Chart{}, ErrAsk\n'
assert old in s, 'm446'
p.write_text(s.replace(old, '\tif ask.Origin.Lat < -90 || ask.Origin.Lat > 90 {\n\t\treturn Chart{}, ErrAsk\n'))
