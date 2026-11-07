import pathlib
# 0.19.5 Q5.4 (D-17): the 16 abbreviations have one table, which names every
# point as the three copies it replaced did.
# Killed by TestCompass16MatchesTheTablesItReplaces.
p = pathlib.Path('platform/geo/geo.go'); s = p.read_text()
old = '"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}\n\treturn pts[CompassIndex(deg, 16)]'
assert s.count(old) == 1, 'm765'
p.write_text(s.replace(old, '"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SWS", "SW", "WSW", "W", "WNW", "NW", "NNW"}\n\treturn pts[CompassIndex(deg, 16)]', 1))
