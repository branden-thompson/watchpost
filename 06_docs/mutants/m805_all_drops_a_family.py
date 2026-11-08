import pathlib
# 0.19.5 M5 (the provider key): all leaves the seismic family out, so neither
# the live pipelines nor the one-shot report asks usgs.
# Killed by TestAllAsksEveryFamily.
p = pathlib.Path('app/dashboard.go'); s = p.read_text()
old = '\treturn append(out, f.seismic...)\n'
assert s.count(old) == 1, 'm805'
p.write_text(s.replace(old, '\treturn out\n', 1))
