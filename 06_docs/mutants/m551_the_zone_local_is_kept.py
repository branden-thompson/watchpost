import pathlib
# 0.19.0 R2I-2: the geocoder's zone "Local" is kept, giving the place the listener's own clock.
# Killed by TestTheGeocodersTextIsCleaned.
p = pathlib.Path('domains/locations/openmeteo/geocode.go'); s = p.read_text()
old = ' || r.Timezone == "Local" {'
assert s.count(old) == 1, 'm551'
p.write_text(s.replace(old, ' {', 1))
