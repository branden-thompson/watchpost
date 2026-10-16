import pathlib
# 0.19.0 FR-3.1: the next turn is where the sun crosses the horizon.
# Killed by TestTheNextTurnIsTheSunriseOrSunset.
p = pathlib.Path('platform/astro/turn.go'); s = p.read_text()
old = '\t\tif Up(lat, lon, to) == up {\n'
assert old in s, 'm351'
p.write_text(s.replace(old, '\t\tif Up(lat, lon, to) != up {\n'))
