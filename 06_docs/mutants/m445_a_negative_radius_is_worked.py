import pathlib
# 0.19.0 D-212: an ask with a radius negative or not a number is refused.
# Killed by TestAnAskOffTheGlobeIsRefused.
p = pathlib.Path('platform/propanswers/answers.go'); s = p.read_text()
old = '\tif !(ask.RadiusKm >= 0) {\n\t\treturn Chart{}, ErrAsk\n'
assert old in s, 'm445'
p.write_text(s.replace(old, '\tif ask.RadiusKm < -1e9 {\n\t\treturn Chart{}, ErrAsk\n'))
