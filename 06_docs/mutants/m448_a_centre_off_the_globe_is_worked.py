import pathlib
# 0.19.0 D-212: a centre off the globe is refused.
# Killed by TestAnAskOffTheGlobeIsRefused.
p = pathlib.Path('platform/propanswers/answers.go'); s = p.read_text()
old = '\tif !(centre.Lat >= -90 && centre.Lat <= 90) {\n\t\treturn Centre{}, ErrAsk\n'
assert old in s, 'm448'
p.write_text(s.replace(old, '\tif false {\n\t\treturn Centre{}, ErrAsk\n'))
