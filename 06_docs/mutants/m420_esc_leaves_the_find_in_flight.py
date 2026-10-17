import pathlib
# 0.19.0 AX-11: Esc cancels a find still being looked up.
# Killed by TestEscCancelsAFindInFlight.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\t\t\td.mapPane.propFindSeq++ // the find in flight cancelled'
assert old in s, 'm420'
p.write_text(s.replace(old, '\t\t\t// the find in flight cancelled'))
