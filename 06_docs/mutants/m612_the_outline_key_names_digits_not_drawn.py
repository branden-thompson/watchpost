import pathlib
# 0.19.3 AX-8 (A-107 (9)): the badge keys only the digits the picture draws.
# Killed by TestTheOutlineDigitsAreKeyed.
p = pathlib.Path('modes/tty/map_scrub.go'); s = p.read_text()
old = '\t\tif d.mapPane.drawnSev[strings.ToLower(s.Word())] {\n'
assert s.count(old) == 1, 'm612'
p.write_text(s.replace(old, '\t\tif d.mapPane.drawnSev[strings.ToLower(s.Word())] || true {\n', 1))
