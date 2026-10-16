import pathlib
# 0.19.0 FR-3.1 (AX-E): the sun up above the horizon.
# Killed by TestTheNextTurnIsTheSunriseOrSunset.
p = pathlib.Path('platform/astro/turn.go'); s = p.read_text()
old = 'func Up(lat, lon float64, at time.Time) bool { return Elevation(lat, lon, at) > horizon }'
assert old in s, 'm349'
p.write_text(s.replace(old, 'func Up(lat, lon float64, at time.Time) bool { return Elevation(lat, lon, at) < horizon }'))
