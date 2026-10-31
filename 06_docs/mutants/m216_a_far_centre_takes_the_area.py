import pathlib
# 0.19.0 FR-1.11: a centre past the near-vertical radius is answered by its path.
# Killed by TestTheCentreIsAnsweredAsTheMapPans.
p = pathlib.Path('platform/propanswers/answers.go'); s = p.read_text()
old = '\tif c.Km <= radiusKm {\n'
assert old in s, 'm216'
p.write_text(s.replace(old, '\tif c.Km >= 0 {\n'))
