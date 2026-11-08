import pathlib
# 0.19.5 M5 (the provider key): seismic loses its domainFor case, so Serves is
# false for usgs and quakes are fetched nowhere.
# Killed by TestEveryFetchKindHasADomain.
p = pathlib.Path('platform/sched/sched.go'); s = p.read_text()
old = '\tcase snapshot.KindSeismic:\n\t\treturn "seismic"\n'
assert s.count(old) == 1, 'm804'
p.write_text(s.replace(old, '', 1))
