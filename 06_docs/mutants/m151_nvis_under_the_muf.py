import pathlib
# D-157: the near-vertical bands are those under foF2.
# Killed by TestThePropagationRowsFollowTheMock.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tout = append(out, "  Local, to ~400 km (NVIS): "+bandNames(bandsUnder(fo)))'
assert old in s, 'm151'
p.write_text(s.replace(old, '\tout = append(out, "  Local, to ~400 km (NVIS): "+bandNames(bandsUnder(muf)))'))
