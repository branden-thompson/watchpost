import pathlib
# 0.19.0 R2I-3: the tile transport reads a reply's headers up to Go's own ten megabytes.
# Killed by TestTheTileTransportCapsItsHeaders.
p = pathlib.Path('app/maptiles.go'); s = p.read_text()
old = ' MaxResponseHeaderBytes: tileMaxHeaderBytes,'
assert s.count(old) == 1, 'm552'
p.write_text(s.replace(old, '', 1))
