import pathlib
# 0.19.0 FR-4.9: the geocoder's name parameter is redacted.
# Killed by TestTheGeocoderQueryIsRedacted.
p = pathlib.Path('platform/httpx/httpx.go'); s = p.read_text()
old = '\tcase "name": // the geocoder'
assert old in s, 'm316'
p.write_text(s.replace(old, '\tcase "name-unredacted": // the geocoder'))
