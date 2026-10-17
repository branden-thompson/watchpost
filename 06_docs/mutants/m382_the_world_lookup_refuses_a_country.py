import pathlib
# 0.19.0 FR-1.10 (D-61): the world lookup takes a place in any country.
# Killed by TestTheOnlineFallbackIsUncached.
p = pathlib.Path('domains/locations/openmeteo/geocode.go'); s = p.read_text()
old = '\tif !world && r.CountryCode != "" && !coverage.NWS(r.CountryCode) {'
assert old in s, 'm382'
p.write_text(s.replace(old, '\tif r.CountryCode != "" && !coverage.NWS(r.CountryCode) {'))
