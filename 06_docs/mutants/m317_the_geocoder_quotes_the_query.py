import pathlib
# 0.19.0 FR-4.9: no error quotes what was typed.
# Killed by TestTheGeocoderQueryIsRedacted.
p = pathlib.Path('domains/locations/openmeteo/geocode.go'); s = p.read_text()
old = 'fmt.Errorf("geocoding: %w", err)'
assert old in s, 'm317'
p.write_text(s.replace(old, 'fmt.Errorf("geocoding %q: %w", query, err)'))
