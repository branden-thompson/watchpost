import pathlib
# 0.19.0 FR-3.1: the sun's hour angle follows the longitude.
# Killed by TestTheSunIsUpByDayAndDownByNight.
p = pathlib.Path('platform/astro/turn.go'); s = p.read_text()
old = '\ttrueSolar := hour*60 + eqTime + 4*lon // minutes\n'
assert old in s, 'm350'
p.write_text(s.replace(old, '\ttrueSolar := hour*60 + eqTime // minutes\n'))
