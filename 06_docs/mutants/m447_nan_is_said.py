import pathlib
# 0.19.0 D-212: a figure that is not a number is never said "NaN".
# Killed by TestAnAskOffTheGlobeIsRefused.
p = pathlib.Path('platform/propanswers/limits.go'); s = p.read_text()
old = '\tif v != v {\n\t\treturn "?"'
assert old in s, 'm447'
p.write_text(s.replace(old, '\tif false {\n\t\treturn "?"'))
