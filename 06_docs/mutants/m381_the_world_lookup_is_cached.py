import pathlib
# 0.19.0 FR-1.10: the Propagation mode's online lookup keeps no answer.
# Killed by TestTheOnlineFallbackIsUncached.
p = pathlib.Path('domains/locations/openmeteo/geocode.go'); s = p.read_text()
old = '\tif world {\n\t\topts = append(opts, httpx.NoCache()) // never kept (FR-1.10)\n'
assert old in s, 'm381'
p.write_text(s.replace(old, '\tif false && world {\n\t\topts = append(opts, httpx.NoCache()) // never kept (FR-1.10)\n'))
