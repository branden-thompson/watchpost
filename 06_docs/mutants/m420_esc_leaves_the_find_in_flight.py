import pathlib
# 0.19.0 AX-11: Esc cancels a find still being looked up.
# Killed by TestEscCancelsAFindInFlight.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\td.mapPane.propFindSeq++\n\td.mapPane.propFindBusy = false\n'
assert s.count(old) == 1, 'm420'
p.write_text(s.replace(old, '\td.mapPane.propFindBusy = false\n', 1))
