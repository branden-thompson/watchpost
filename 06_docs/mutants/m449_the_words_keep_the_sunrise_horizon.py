import pathlib
# 0.19.0 L-F9: the words say day or night at the horizon the drawn terminator uses.
# Killed by TestDayAndNightAreSaidAsTheLineIsDrawn.
p = pathlib.Path('platform/astro/turn.go'); s = p.read_text()
old = 'const horizon = 0.0'
assert old in s, 'm449'
p.write_text(s.replace(old, 'const horizon = -0.833'))
