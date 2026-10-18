import pathlib
# 0.19.0 BUILD red team round 2 (FR-4.9, PN2-6): the geocoder's text is cleaned of escapes and controls.
# Killed by TestTheGeocodersTextIsCleaned.
p = pathlib.Path('domains/locations/openmeteo/geocode.go'); s = p.read_text()
old = 'func cleanField(s string) string { return plaintext.ClampField(plaintext.Line(s)) }'
assert old in s, 'm481'
p.write_text(s.replace(old, 'func cleanField(s string) string { return plaintext.ClampField(s) }', 1))
