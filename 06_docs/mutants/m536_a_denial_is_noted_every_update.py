import pathlib
# 0.19.0 D-247: the diagnostics name GIRO's denial on every update, not once as it begins.
# Killed by TestTheLibrarysRealStatesAreSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'newlyDenied := snap.Inputs.GIRO == ionomaps.GIRODenied && (d.mapPane.propSnap == nil || d.mapPane.propSnap.Inputs.GIRO != ionomaps.GIRODenied)'
assert s.count(old) == 1, 'm536'
p.write_text(s.replace(old, 'newlyDenied := snap.Inputs.GIRO == ionomaps.GIRODenied', 1))
