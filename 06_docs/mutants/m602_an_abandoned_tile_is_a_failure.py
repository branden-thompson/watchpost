import pathlib
# 0.19.1 D-354: a tile the map gave up is a try and not the host failing.
# Killed by TestAnAbandonedTileIsNotAFailure and TestATileTheMapGivesUpIsNotAFailure.
p = pathlib.Path('app/maptiles.go'); s = p.read_text()
old = '\tcase err != nil && httpx.Abandoned(req.Context()):\n'
assert s.count(old) == 1, 'm602'
p.write_text(s.replace(old, '\tcase err != nil && httpx.Abandoned(req.Context()) && false:\n', 1))
