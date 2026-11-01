import pathlib
# 0.19.1 D-354: the tile counter refuses a transport that gives neither a reply nor an error.
# Killed by TestATransportGivingNothingBackIsRefused.
p = pathlib.Path('app/maptiles.go'); s = p.read_text()
old = '\tif bad := invariant.Check(res != nil || err != nil, "maptiles: the transport gave neither a reply nor an error"); bad != nil {\n'
assert s.count(old) == 1, 'm606'
p.write_text(s.replace(old, '\tif bad := invariant.Check(res != nil || err != nil || true, "maptiles: the transport gave neither a reply nor an error"); bad != nil {\n', 1))
