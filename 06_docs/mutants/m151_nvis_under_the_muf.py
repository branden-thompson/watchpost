import pathlib
# D-157: the near-vertical bands are those under foF2.
# Killed by TestThePropagationRowsFollowTheMock.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tout = append(out, under+"Local, to ~"+strconv.Itoa(d.propRadiusKm())+" km ("+d.propLocalName()+"): "+bandNames(bandsUnder(fo)))'
assert old in s, 'm151'
p.write_text(s.replace(old, '\tout = append(out, under+"Local, to ~"+strconv.Itoa(d.propRadiusKm())+" km ("+d.propLocalName()+"): "+bandNames(bandsUnder(muf)))'))
